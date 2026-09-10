#!/usr/bin/env python3
"""Refresh the public format tables. Review the snapshot and mapping after each refresh."""

from datetime import datetime, timezone
import hashlib
from html.parser import HTMLParser
import json
from pathlib import Path
from urllib.request import urlopen


class Tables(HTMLParser):
    def __init__(self):
        super().__init__()
        self.rows = []
        self.row = None
        self.cell = None

    def handle_starttag(self, tag, attrs):
        if tag == "tr":
            self.row = []
        if tag in ("td", "th") and self.row is not None:
            self.cell = ""

    def handle_data(self, text):
        if self.cell is not None:
            self.cell += text

    def handle_endtag(self, tag):
        if tag in ("td", "th") and self.cell is not None:
            self.row.append(self.cell.strip())
            self.cell = None
        if tag == "tr" and self.row is not None:
            self.rows.append(self.row)
            self.row = None


def main():
    result = {"fetched_at": datetime.now(timezone.utc).isoformat(), "pages": []}
    for version in ("3-5", "6", "7", "8"):
        url = f"https://alphatab.net/docs/formats/guitar-pro-{version}"
        with urlopen(url, timeout=30) as response:
            html = response.read()
        parser = Tables()
        parser.feed(html.decode())
        rows, domain = [], ""
        for row in parser.rows:
            if len(row) != 6 or row[0] == "Feature":
                continue
            if not any(row[1:]):
                domain = row[0]
                continue
            rows.append({"domain": domain, "feature": row[0].removeprefix("⭐ "),
                         "model": row[1], "reading": row[2], "rendering": row[3],
                         "audio": row[4], "alphaTex": row[5]})
        if not rows:
            raise ValueError(f"No feature table found at {url}")
        result["pages"].append({"url": url, "format": f"gp{version}",
                                "html_sha256": hashlib.sha256(html).hexdigest(), "rows": rows})
    path = Path(__file__).with_name("website-features.json")
    path.write_text(json.dumps(result, indent=2, ensure_ascii=False) + "\n")
    print(f"Saved {sum(len(p['rows']) for p in result['pages'])} format rows")


if __name__ == "__main__":
    main()
