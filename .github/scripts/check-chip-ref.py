import re
import sys
from pathlib import Path

import yaml

branch = sys.argv[1]
part = yaml.safe_load(Path("snap/snapcraft.yaml").read_text())["parts"]["connectedhomeip"]
refs = {key: part[key] for key in ("source-tag", "source-branch", "source-commit") if key in part}
if branch == "main" and set(refs) == {"source-tag"} and re.fullmatch(r"v\d+(\.\d+){3}", refs["source-tag"]):
    print(refs["source-tag"])
elif branch == "dev" and refs == {"source-branch": "master"}:
    print("master")
else:
    sys.exit(f"Unexpected connectedhomeip source on {branch}: {refs}")
