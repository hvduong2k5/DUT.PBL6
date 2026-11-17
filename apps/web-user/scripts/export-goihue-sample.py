"""Export only public product/category records from a PostgreSQL COPY-text backup."""
import argparse
import hashlib
import json
import re
from pathlib import Path

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("sql", type=Path)
parser.add_argument("--output", type=Path, default=Path(__file__).resolve().parents[1] / "src/data/goihue-sample.json")
args = parser.parse_args()
raw = args.sql.read_bytes()
text = raw.decode("utf-8")
match = re.search(r"COPY public.records \((.*?)\) FROM stdin;\r?\n(.*?)\r?\n\\\.", text, re.S)
if match is None:
    raise SystemExit("Missing COPY public.records block.")
columns = match[1].split(", ")
result = {"source": args.sql.name, "sourceSha256": hashlib.sha256(raw).hexdigest(), "products": [], "categories": []}
fields = {
    "product": ["id", "name", "slug", "image", "price", "stock", "active", "category", "description"],
    "category": ["id", "code", "name", "image", "active", "sortOrder", "description"],
}
escapes = {"\\": "\\", "t": "\t", "n": "\n", "r": "\r"}
for line in match[2].splitlines():
    row = dict(zip(columns, line.split("\t")))
    kind = row.get("kind")
    if kind not in fields:
        continue
    payload = re.sub(r"\\(\\|t|n|r)", lambda m: escapes[m[1]], row["data"])
    record = json.loads(payload)
    destination = "products" if kind == "product" else "categories"
    result[destination].append({key: record[key] for key in fields[kind] if key in record})
if not result["products"] or not result["categories"]:
    raise SystemExit("Backup has no product/category records.")
args.output.parent.mkdir(parents=True, exist_ok=True)
args.output.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print(f"Exported {len(result['products'])} products and {len(result['categories'])} categories.")
