"""Normalize one advertising campaign export for the Marketing CSV API."""

import argparse
import csv
import re
import sys
from datetime import date
from pathlib import Path
from zoneinfo import ZoneInfo, ZoneInfoNotFoundError

SOURCE_HEADER = ["campaign_id", "date", "impressions", "clicks"]
TARGET_HEADER = ["external_execution_key", "date", "source_version", "impressions", "clicks"]
MAX_INTEGER = 9_007_199_254_740_991


def integer(value, minimum):
    if not re.fullmatch(r"[0-9]{1,16}", value):
        raise ValueError("Expected a nonnegative integer")
    number = int(value)
    if not minimum <= number <= MAX_INTEGER:
        raise ValueError("Integer is outside the supported range")
    return number


def normalize(path, source, account, campaign, version):
    for value in (source, account, campaign):
        if not re.fullmatch(r"[A-Za-z0-9._-]{1,52}", value):
            raise ValueError("Source, account and campaign must be compact ASCII identifiers")
    key = f"{source}:{account}:{campaign}"
    rows = {}
    with path.open(encoding="utf-8-sig", newline="") as stream:
        reader = csv.reader(stream, strict=True)
        if next(reader, None) != SOURCE_HEADER:
            raise ValueError("Unexpected export header")
        for row in reader:
            if len(row) != len(SOURCE_HEADER):
                raise ValueError(f"Unexpected number of columns at line {reader.line_num}")
            if row[0] != campaign:
                continue
            try:
                if not re.fullmatch(r"[0-9]{4}-[0-9]{2}-[0-9]{2}", row[1]):
                    raise ValueError("Date must be YYYY-MM-DD")
                day = date.fromisoformat(row[1])
                if day in rows:
                    raise ValueError("Duplicate campaign date")
                rows[day] = [key, day.isoformat(), version, integer(row[2], 0), integer(row[3], 0)]
                if len(rows) > 31:
                    raise ValueError("Export more than 31 campaign days in separate files")
            except ValueError as error:
                raise ValueError(f"Line {reader.line_num}: {error}") from error
    if not rows:
        raise ValueError("The export has no rows for the requested campaign")
    return [rows[day] for day in sorted(rows)]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("export", type=Path)
    parser.add_argument("--source", required=True)
    parser.add_argument("--account", required=True)
    parser.add_argument("--campaign", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--export-timezone", required=True)
    parser.add_argument("--execution-timezone", required=True)
    args = parser.parse_args()
    try:
        ZoneInfo(args.export_timezone)
        if args.export_timezone != args.execution_timezone:
            raise ValueError("Export timezone must match the execution timezone")
        rows = normalize(args.export, args.source, args.account, args.campaign, integer(args.version, 1))
    except (OSError, ValueError, ZoneInfoNotFoundError, csv.Error) as error:
        parser.exit(2, f"Export rejected: {error}\n")
    writer = csv.writer(sys.stdout, lineterminator="\n")
    writer.writerow(TARGET_HEADER)
    writer.writerows(rows)


if __name__ == "__main__":
    main()
