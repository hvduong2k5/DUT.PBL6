import { request as httpRequest } from "node:http";
import { request as httpsRequest } from "node:https";
import { Readable, Transform } from "node:stream";
import { pipeline } from "node:stream/promises";
import { NextRequest, NextResponse } from "next/server";
import { fulfillmentFailure, fulfillmentUpstream } from "./fulfillment-handler";

const maximum = 209715200;
// Only URLs issued by the authenticated provider, on an explicitly configured
// storage origin, may receive bytes. Neither a URL nor an object key is accepted
// from the browser. Server relay also avoids browser CORS/private Docker DNS.
export function storageUploadTarget(raw: string) {
  const url = new URL(raw);
  const allowed = new URL(process.env.FULFILLMENT_STORAGE_UPLOAD_ORIGIN ?? "");
  if (url.origin !== allowed.origin || !["http:", "https:"].includes(url.protocol) || url.username || url.password || url.hash) throw new Error("Untrusted upload origin");
  const connect = process.env.FULFILLMENT_STORAGE_UPLOAD_CONNECT_ORIGIN;
  const target = connect ? new URL(connect) : url;
  if (!["http:", "https:"].includes(target.protocol) || target.username || target.password || (connect && (target.pathname !== "/" || target.search || target.hash))) throw new Error("Invalid storage connection origin");
  return { url, target };
}
export async function uploadPackingVideo(request: NextRequest, taskPath: string) {
  const size = Number(request.headers.get("x-video-size"));
  const hash = request.headers.get("x-video-sha256") ?? "";
  const type = request.headers.get("content-type") ?? "";
  const key = request.headers.get("idempotency-key") ?? "";
  const packageId = request.headers.get("x-package-id") ?? "";
  if (!Number.isSafeInteger(size) || size <= 0 || size > maximum || !/^[0-9a-f]{64}$/u.test(hash)
    || !["video/mp4", "video/webm"].includes(type) || key.length < 16 || key.length > 128
    || !/^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/u.test(packageId) || !request.body) return fulfillmentFailure("INVALID_MEDIA", 400);
  const session = await fulfillmentUpstream(request, taskPath + "/media-sessions", "POST", JSON.stringify({ package_id: packageId, media_type: type, size_bytes: size, sha256: hash }), key);
  if (!session.ok) return new NextResponse(session.body, { status: session.status, headers: { "Content-Type": "application/json", "Cache-Control": "no-store" } });
  const media = await session.json() as { upload_url: string; file_id: string; status: string };
  const { url, target } = storageUploadTarget(media.upload_url);
  let received = 0;
  const bounded = new Transform({ transform(chunk: Buffer, _encoding, callback) {
    received += chunk.length;
    callback(received > size ? new Error("Video exceeds declared size") : null, received > size ? undefined : chunk);
  }, flush(callback) { callback(received === size ? undefined : new Error("Video length mismatch")); } });
  const upload = (target.protocol === "https:" ? httpsRequest : httpRequest)({
    hostname: target.hostname, port: target.port, path: url.pathname + url.search, method: "PUT",
    headers: { Host: url.host, "Content-Type": type, "Content-Length": size },
  });
  upload.setTimeout(120000, () => upload.destroy(new Error("Upload timeout")));
  const abort = () => upload.destroy(new Error("Upload cancelled"));
  request.signal.addEventListener("abort", abort, { once: true });
  const result = new Promise<void>((resolve, reject) => {
    upload.on("error", reject);
    upload.on("response", (response) => {
      response.resume();
      response.on("end", () => (response.statusCode && response.statusCode >= 200 && response.statusCode < 300) ? resolve() : reject(new Error("Storage rejected upload")));
      response.on("error", reject);
    });
  });
  try {
    await Promise.all([pipeline(Readable.fromWeb(request.body as never), bounded, upload), result]);
  } finally { request.signal.removeEventListener("abort", abort); }
  return NextResponse.json({ file_id: media.file_id, status: media.status }, { status: 201, headers: { "Cache-Control": "no-store" } });
}
