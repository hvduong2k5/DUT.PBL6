"""Generate Traceability gRPC clients/server stubs from the approved contracts.

Requires grpcio-tools==1.85.4, protoc-gen-go v1.36.6 and protoc-gen-go-grpc v1.5.1 on PATH.
Inventory field-9 still uses its approved candidate; canonical Catalog/Media and two Traceability workflow RPCs use
canonical integration-target files. This command only writes service-local Go.
"""
from pathlib import Path
import argparse
import os
import subprocess
import sys
import tempfile
import grpc_tools

SERVICE = Path(__file__).resolve().parents[1]
ROOT = SERVICE.parents[1]
SOURCES = {
    "traceability/v1/batch_source.proto": ROOT / "docs/03_api_specs/traceability-workflow-candidates/batch_source.proto",
    "traceability/v1/batch_links.proto": ROOT / "packages/proto/traceability/v1/batch_links.proto",
    "catalog/v1/catalog.proto": ROOT / "packages/proto/catalog/v1/catalog.proto",
    "common/v1/types.proto": ROOT / "packages/proto/common/v1/types.proto",
    "common/v1/errors.proto": ROOT / "packages/proto/common/v1/errors.proto",
    "media/v1/media_service.proto": ROOT / "packages/proto/media/v1/media_service.proto",
    "identity/v1/staff_authorization.proto": ROOT / "packages/proto/identity/v1/staff_authorization.proto",
}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="verify generated clients without changing the worktree")
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix="pbl6-trace-batch-codegen-") as directory:
        root = Path(directory)
        targets = []
        for relative, source in SOURCES.items():
            target = root / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text(source.read_text(encoding="utf-8"), encoding="utf-8")
            targets.append(str(target))
        output = root / "output"
        output.mkdir()
        env = os.environ.copy()
        env["PATH"] = str(Path.home() / "go/bin") + os.pathsep + env["PATH"]
        subprocess.run([
            sys.executable, "-m", "grpc_tools.protoc", "-I" + str(root),
            "-I" + str(Path(grpc_tools.__file__).parent / "_proto"),
            "--go_out=" + str(output), "--go_opt=paths=source_relative",
            "--go-grpc_out=" + str(output), "--go-grpc_opt=paths=source_relative",
            "--go_opt=Mcommon/v1/types.proto=github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/common/v1",
            "--go_opt=Mcommon/v1/errors.proto=github.com/hvduong2k5/DUT.PBL6/services/traceability-service/pkg/proto/common/v1", *targets,
        ], cwd=SERVICE, env=env, check=True)
        for relative in SOURCES:
          for suffix in ([".pb.go", "_grpc.pb.go"] if "service " in SOURCES[relative].read_text(encoding="utf-8") else [".pb.go"]):
            generated_path = Path(relative.removesuffix(".proto") + suffix)
            generated = (output / generated_path).read_bytes().replace(b"\r\n", b"\n")
            destination = SERVICE / "pkg/proto" / generated_path
            if args.check:
                if destination.read_bytes().replace(b"\r\n", b"\n") != generated:
                    raise SystemExit(f"Generated client mismatch: {destination.relative_to(SERVICE)}")
            else:
                destination.parent.mkdir(parents=True, exist_ok=True)
                destination.write_bytes(generated)
    print(("Verified" if args.check else "Generated") + f" {len(SOURCES)} Traceability-local proto files; canonical provider methods unchanged.")


if __name__ == "__main__":
    main()
