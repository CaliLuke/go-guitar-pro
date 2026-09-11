#!/usr/bin/env python3
"""Refresh exact upstream review identities without accepting new constructs."""

import collections
import fnmatch
import json
from pathlib import Path
import re

import manage


PATH = manage.HERE / "upstream-ownership.json"


def policy_for(policies, path):
    for policy in policies:
        if fnmatch.fnmatchcase(path, policy["pattern"]):
            return policy
    raise ValueError(f"{path} has no upstream path policy")


def refresh():
    data = json.loads(PATH.read_text())
    sources, constructs = manage.discover()
    declarations = {c["name"]: c for c in constructs
                    if c["kind"] in ("field", "enum-member") and "/model/" in c["path"]}
    existing = {review["construct_id"]: review for review in data["construct_reviews"]}
    required = []
    for construct in constructs:
        policy = policy_for(data["path_policies"], construct["path"])
        if construct["kind"] not in policy["review_kinds"]:
            continue
        prior = existing.get(construct["id"])
        if prior and prior["disposition"] == "authored" and not prior.get("model_declaration"):
            inferred = manage.infer_assignment_model_declaration(construct, declarations)
            if inferred:
                prior["model_declaration"] = inferred
        required.append(prior if prior else {
            "construct_id": construct["id"], "source_scope": policy["source_scope"], "formats": policy["formats"],
            "disposition": "unclassified", "reason": "", "primary_capability": None,
            "secondary_capabilities": [], "model_declaration": None,
        })
    active_reviews = list(required)
    matched_ids = {review["construct_id"] for review in required}
    required.extend(review for construct_id, review in existing.items() if construct_id not in matched_ids)
    data["construct_reviews"] = sorted(required, key=lambda review: review["construct_id"])
    construct_by_id = {construct["id"]: construct for construct in constructs}
    declaration_sources = collections.defaultdict(list)
    for review in active_reviews:
        if review["disposition"] != "authored" or not review.get("primary_capability"):
            continue
        if review.get("model_declaration"):
            declaration_sources[review["model_declaration"]].append(review)
        source_line = construct_by_id[review["construct_id"]]["declaration"]
        for owner, member in re.findall(r"\b([A-Z][_$A-Za-z0-9]*)\.([_$A-Za-z][_$A-Za-z0-9]*)\b", source_line):
            declaration = f"{owner}.{member}"
            if declaration in declarations:
                declaration_sources[declaration].append(review)
    previous_model = {review["declaration"]: review for review in data.get("model_reviews", [])}
    model_reviews = []
    for declaration in declaration_sources:
        prior = previous_model.get(declaration)
        if prior:
            model_reviews.append(prior)
            continue
        model_reviews.append({
            "declaration": declaration,
            "formats": [], "disposition": "unclassified", "reason": "",
            "primary_capability": None, "secondary_capabilities": [],
        })
    matched_declarations = {review["declaration"] for review in model_reviews}
    model_reviews.extend(review for declaration, review in previous_model.items()
                         if declaration not in matched_declarations)
    data["model_reviews"] = sorted(model_reviews, key=lambda review: review["declaration"])
    PATH.write_text(json.dumps(data, indent=2) + "\n")


if __name__ == "__main__":
    refresh()
