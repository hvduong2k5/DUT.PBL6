"""Real Fulfillment, versioned RustFS storage, ffprobe and ClamAV acceptance.

The isolated scanner loads a tiny custom test database, not production AV
signatures. No mock scan verdicts, fake revisions or database verification flags.
"""
import base64
import hashlib
import json
import secrets
import socket
import subprocess
import time
import urllib.parse
import urllib.request
import uuid

RUSTFS = "rustfs/rustfs@sha256:ed9210d6411d77ffceb70d94b964a3cb7d225983368bc25bebe01e995293dcc2"
CLAMAV = "clamav/clamav@sha256:58168cd4f48a118c7d20b76da8ffce97e579b84ed7e6d1e977c6e22b1e989c5a"
IMAGE = "omamx/fulfillment-service:integration"


def prepare(root, folder, ports, env, accounts, sql, run, container, request,
            issuer, identity, db_name, password, suffix, peer_address):
    for name in ["order-reads", "fulfillment-reads"]:
        (folder / name).write_text(secrets.token_urlsafe(40))
    env.update(
        ORDER_FULFILLMENT_GRPC=f"127.0.0.1:{ports['fulfillment_grpc']}",
        ORDER_FULFILLMENT_TOKEN_FILE=str(folder / "fulfillment-reads"),
        ORDER_FULFILLMENT_CALLER_TOKEN_FILE=str(folder / "order-reads"),
        ORDER_FULFILLMENT_ALLOW_LOCAL_INSECURE="true",
        ORDER_PACKING_POLICY_FILE=str(folder / "packing-policy.json"),
        WEB_TEST_ORDER_GRPC_PORT=str(ports["order_grpc"]),
    )
    (folder / "packing-policy.json").write_text(json.dumps({
        "policy_version": "mvp-video-1", "evidence_mode": "VIDEO_REQUIRED",
        "checklist": [{"item_code": "ITEMS_MATCH", "definition": "Count matches the committed lot", "required": True}],
        "maximum_weight_grams": 30000, "maximum_dimension_cm": 200,
        "maximum_video_bytes": "209715200", "maximum_video_seconds": 120,
    }))
    # Owner's seeded ADMIN role, assigned to this disposable Employee only.
    sql(f"INSERT INTO identity.iam_role_assignments(assignment_id,principal_id,role_id,scope_type) SELECT '{uuid.uuid4()}','{accounts['E']['principal']}',role_id,'GLOBAL' FROM identity.iam_roles WHERE role_code='ADMIN';")
    public = request(issuer)[1]["public_key"]
    (folder / "jwt-public.pem").write_text("-----BEGIN PUBLIC KEY-----\n" + "\n".join(public[i:i+64] for i in range(0,len(public),64)) + "\n-----END PUBLIC KEY-----\n")
    access, secret = secrets.token_urlsafe(24), secrets.token_urlsafe(40)
    storage = "codex-web-storage-" + suffix
    container(storage, RUSTFS, ["-e", "RUSTFS_ACCESS_KEY="+access, "-e", "RUSTFS_SECRET_KEY="+secret,
        "-e", "RUSTFS_VOLUMES=/data", "-p", f"127.0.0.1:{ports['storage']}:9000"])
    for _ in range(120):
        try:
            with socket.create_connection(("127.0.0.1",ports['storage']),1): break
        except OSError: time.sleep(.5)
    else: raise RuntimeError("RustFS did not start")
    previous = env.copy()
    env.update(S3_ENDPOINT=f"127.0.0.1:{ports['storage']}",S3_ACCESS_KEY=access,S3_SECRET_KEY=secret,S3_BUCKET="packing-evidence")
    run([env.get("MVP_GO", "go"),"run","./tests/storagefixture"],root/"services/fulfillment-service")
    env.clear();env.update(previous)
    marker = "OMAMA-FIXTURE-BLOCK"
    (folder / "fixture.ndb").write_text("OmamaFixtureMarker:0:*:"+marker.encode().hex()+"\n",newline="\n")
    (folder / "clamd.conf").write_text("Foreground yes\nUser root\nDatabaseDirectory /fixture-db\nTCPSocket 3310\nTCPAddr 0.0.0.0\nStreamMaxLength 210M\nLogTime yes\n",newline="\n")
    scanner = "codex-web-scanner-"+suffix
    # Bypass image entrypoint/freshclam: deterministic custom signatures only.
    container(scanner, CLAMAV, ["--entrypoint", "clamd", "-v", str(folder / "fixture.ndb")+":/fixture-db/fixture.ndb:ro",
        "-v",str(folder / "clamd.conf")+":/etc/clamav/clamd.conf:ro", "-p",f"127.0.0.1:{ports['scanner']}:3310"])
    for _ in range(90):
        try:
            with socket.create_connection(("127.0.0.1",ports['scanner']),1) as s:
                s.sendall(b"zPING\0")
                if s.recv(64).startswith(b"PONG"): break
        except OSError: pass
        time.sleep(.5)
    else: raise RuntimeError("Real ClamAV not ready")
    # Use container DNS on an isolated network for storage/scanner; host-facing
    # listeners remain loopback. All names belong to this fixture and cleanup.
    network = "codex-web-fulfillment-"+suffix
    run(["docker","network","create",network],stdout=subprocess.DEVNULL)
    for name in [storage,scanner]:run(["docker","network","connect",network,name],stdout=subprocess.DEVNULL)
    sql("CREATE DATABASE fulfillment_test;")
    dsn=f"postgres://peer:{password}@host.docker.internal:{ports['db']}/fulfillment_test?sslmode=disable"
    run(["docker","run","--rm","--add-host","host.docker.internal:host-gateway","-e","FULFILLMENT_DATABASE_URL="+dsn,
         "--entrypoint","fulfillment-migrate",IMAGE],stdout=subprocess.DEVNULL)
    values={
        "FULFILLMENT_ENV":"local","FULFILLMENT_DATABASE_URL":dsn,
        "FULFILLMENT_JWT_ISSUER":issuer,"FULFILLMENT_JWT_PUBLIC_KEY":"/fixture/jwt-public.pem",
        "FULFILLMENT_STAFF_AUDIENCE":"omama-api", "IDENTITY_URL":f"http://host.docker.internal:{ports['identity']}",
        "FULFILLMENT_ADMISSION_ENABLED":"true","FULFILLMENT_SEAL_ENABLED":"true","FULFILLMENT_PACKING_V2_ENABLED":"true",
        "FULFILLMENT_CANDIDATE_PEERS_ENABLED":"true","FULFILLMENT_MEDIA_UPLOAD_ENABLED":"true","FULFILLMENT_MEDIA_ACCESS_ENABLED":"true",
        "FULFILLMENT_ORDER_GRPC_ENDPOINT":f"host.docker.internal:{ports['order_grpc']}",
        "FULFILLMENT_INVENTORY_GRPC_ENDPOINT":"host.docker.internal:"+peer_address.rsplit(":",1)[1],
        "FULFILLMENT_ORDER_TOKEN_FILE":"/fixture/order-reads","FULFILLMENT_ORDER_CALLER_TOKEN_FILE":"/fixture/fulfillment-reads",
        "FULFILLMENT_INVENTORY_TOKEN_FILE":"/fixture/inventory-reads","FULFILLMENT_INVENTORY_CA_FILE":"/fixture/inventory-ca.pem",
        "FULFILLMENT_INVENTORY_SERVER_NAME":"localhost",
        "FULFILLMENT_STORAGE_ENDPOINT":storage+":9000","FULFILLMENT_STORAGE_BUCKET":"packing-evidence",
        "FULFILLMENT_STORAGE_ACCESS_KEY":access,"FULFILLMENT_STORAGE_SECRET_KEY":secret,
        "FULFILLMENT_CLAMD_ADDR":scanner+":3310", "FULFILLMENT_KAFKA_BROKERS":f"host.docker.internal:{ports['kafka_docker']}",
    }
    extra=["--network",network,"--add-host","host.docker.internal:host-gateway","-v",str(folder)+":/fixture:ro",
        "-p",f"127.0.0.1:{ports['fulfillment']}:8002","-p",f"127.0.0.1:{ports['fulfillment_grpc']}:9002"]
    for k,v in values.items():extra += ["-e",k+"="+v]
    container("codex-web-fulfillment-"+suffix,IMAGE,extra)
    # Actual encoded video, including a second valid MP4 that ClamAV rejects.
    for name,metadata in [("proof.mp4","clean"),("blocked.mp4",marker)]:
        run(["docker","run","--rm","--user","0","--entrypoint","ffmpeg","-v",str(folder)+":/fixture",IMAGE,
            "-v","error","-f","lavfi","-i","color=c=blue:s=160x120:d=1","-metadata","comment="+metadata,
            "-c:v","libx264","-pix_fmt","yuv420p","-y","/fixture/"+name],stdout=subprocess.DEVNULL)
    return network, scanner


def pack(request, gateway, token, customer_token, folder, order_id, order_until,
         shipping, shipping_token, control, run, scanner, shipping_control=None, handoff=None):
    base=gateway+"/api/v1/admin/fulfillment"
    def call(path,body=None,method=None,headers=None):return request(base+path,body,token,method,headers=headers)
    status,denied,_=request(base+"/tasks",token=customer_token)
    assert status==403,("Customer staff gate",status,denied)
    assert request(control+"/pack?order="+order_id,{},method="POST")[0]==409,"TEST packing remained enabled"
    task=None
    for _ in range(60):
        status,page,_=call("/tasks")
        assert status==200,("task queue",status,page)
        task=next((x for x in page["items"] if x["order_id"]==order_id),None)
        if task:break
        time.sleep(1)
    assert task,("Fulfillment did not admit paid order",order_id)
    path="/tasks/"+task["task_id"]
    task=call(path)[1]
    def mutate(action,body,expected=200,key=None):
        nonlocal task
        status,out,_=call(path+"/"+action,body,"PUT" if action.startswith(("picks/","checklist/")) else "POST",
            {"If-Match":'"'+str(task["version"])+'"',"Idempotency-Key":key or uuid.uuid4().hex})
        assert status==expected,(action,status,out)
        if status==200:task=out
        return out
    mutate("assignments",{})
    mutate("start",{})
    for a in task["allocations"]:
        mutate("picks/"+a["allocation_id"],{"counted_quantity":a["allocated_quantity"],"allocation_version":a["allocation_version"]})
    for c in task["checklist"]:
        mutate("checklist/"+c["item_code"],{"result":"PASS"})
    package=task["package"]["package_id"]
    parcel={"weight_grams":1250,"length_cm":30,"width_cm":20,"height_cm":10,"fragile":True}
    def upload(name):
        data=(folder/name).read_bytes()
        status,session,_=call(path+"/media-sessions",{"package_id":package,"media_type":"video/mp4","size_bytes":len(data),"sha256":hashlib.sha256(data).hexdigest()},"POST",{"Idempotency-Key":uuid.uuid4().hex})
        assert status==201,("media session",status,session)
        u=urllib.parse.urlsplit(session["upload_url"])
        # Connect to the published port while retaining the signed Host header.
        url=urllib.parse.urlunsplit((u.scheme,"127.0.0.1:"+str(pack.storage_port),u.path,u.query,u.fragment))
        with urllib.request.urlopen(urllib.request.Request(url,data=data,method="PUT",headers={"Host":u.netloc,"Content-Type":"video/mp4"}),timeout=20) as response:assert response.status==200
        status,out,_=call("/media/"+session["file_id"]+"/complete",{},"POST",{"Idempotency-Key":uuid.uuid4().hex})
        assert status==202,("complete media",status,out)
        return session["file_id"]
    bad=upload("blocked.mp4")
    for _ in range(40):
        badfile=call("/media/"+bad)[1]
        if badfile["status"]=="REJECTED":break
        time.sleep(.5)
    assert badfile["status"]=="REJECTED",("ClamAV failed to reject test signature",badfile)
    mutate("seal",{"package_id":package,"seal_code":"MVP-"+uuid.uuid4().hex.upper(),"parcel":parcel,"primary_evidence_id":bad},409)
    # Scanner outage must leave evidence unverified and prevent sealing.
    run(["docker","stop",scanner],stdout=subprocess.DEVNULL)
    good=upload("proof.mp4")
    time.sleep(2)
    assert call("/media/"+good)[1]["status"]=="VERIFYING"
    seal={"package_id":package,"seal_code":"MVP-"+uuid.uuid4().hex.upper(),"parcel":parcel,"primary_evidence_id":good}
    mutate("seal",seal,409)
    run(["docker","start",scanner],stdout=subprocess.DEVNULL)
    for _ in range(60):
        media=call("/media/"+good)[1]
        if media["status"]=="VERIFIED":break
        time.sleep(1)
    assert media["status"]=="VERIFIED",("Real video verification incomplete",media)
    key=uuid.uuid4().hex
    original_version=task["version"]
    if shipping_control:shipping_control(False)
    mutate("seal",seal,key=key)
    status,replay,_=call(path+"/seal",seal,"POST",{"If-Match":'"'+str(original_version)+'"',"Idempotency-Key":key})
    assert status==200 and replay==task,("seal replay",status,replay)
    original_handoff=None
    if shipping_control:
        for _ in range(30):
            original_handoff=handoff(order_id)
            if original_handoff and original_handoff["attempts"]>0:break
            time.sleep(.5)
        assert original_handoff and original_handoff["attempts"]>0 and original_handoff["shipment_id"] is None and original_handoff["shipments"]==0,("durable Shipping outage",original_handoff)
        shipping_control(True)
    order=order_until(order_id,lambda o:bool(o.get("shipment_id")))
    status,tracking,_=request(shipping+"/api/v1/admin/shipping/shipments/"+order["shipment_id"],token=shipping_token)
    assert status==200,("shipment tracking",status,tracking)
    assert tracking["shipment"]["package_id"]==package and tracking["shipment"]["parcel"]==parcel,("Shipping did not use sealed parcel",tracking)
    if original_handoff:
        recovered=handoff(order_id)
        assert recovered["hash"]==original_handoff["hash"] and recovered["shipments"]==1 and recovered["shipment_id"]==order["shipment_id"],("handoff retry changed request or duplicated shipment",recovered)
    print("PASS: real Fulfillment admission -> staff picks/checklist -> versioned video/ffprobe/ClamAV -> sealed replay -> automatic Order/Shipping; Customer denied, unsafe media rejected, scanner outage blocks seal",flush=True)
    return {"shipment_id":order["shipment_id"],"tracking_code":tracking["shipment"]["tracking_code"]}
