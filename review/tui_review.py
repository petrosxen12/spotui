#!/usr/bin/env python3
"""
TUI review runner — used by both CI (via GitHub Actions) and local eval.

Run from the repository root:
  python review/tui_review.py --bundle <dir> [options]
"""

import argparse
import base64
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile
import urllib.error
import urllib.request


def load_text(path: pathlib.Path) -> str:
    return path.read_text().rstrip()


def load_json(path: pathlib.Path) -> dict:
    return json.loads(path.read_text())


def load_prompt_set(path: pathlib.Path) -> dict:
    return load_json(path)


def compute_prompt_fingerprint(prompt_set: dict, model: str) -> str:
    """SHA-256 of sorted prompt file contents + model name; returns 12-char hex prefix."""
    h = hashlib.sha256()
    for key in sorted(prompt_set["files"]):
        file_path = pathlib.Path(prompt_set["files"][key])
        h.update(key.encode())
        h.update(file_path.read_bytes())
    h.update(model.encode())
    return h.hexdigest()[:12]


def check_rasterizer() -> None:
    if shutil.which("rsvg-convert") is None:
        raise SystemExit(
            "rsvg-convert not found. Install it with:\n"
            "  apt-get install librsvg2-bin  (Debian/Ubuntu)\n"
            "  brew install librsvg          (macOS)"
        )


def rasterize_svgs(bundle_dir: pathlib.Path, tmp_dir: pathlib.Path) -> None:
    """Convert all SVGs in bundle_dir to PNG in tmp_dir."""
    for svg in bundle_dir.glob("*.svg"):
        png = tmp_dir / svg.with_suffix(".png").name
        result = subprocess.run(
            ["rsvg-convert", str(svg), "-o", str(png)],
            capture_output=True,
        )
        if result.returncode != 0:
            raise SystemExit(
                f"rsvg-convert failed for {svg}: {result.stderr.decode(errors='replace')}"
            )


def build_content(
    persona_prompt: str,
    task_directive: str,
    scenario_template: str,
    brief: str,
    manifest: dict,
    bundle_dir: pathlib.Path,
    png_dir: pathlib.Path,
) -> list:
    scenarios = manifest.get("scenarios", [])
    if not scenarios:
        raise SystemExit("manifest.json contains no scenarios")

    content = [
        {
            "type": "input_text",
            "text": "\n\n".join(
                [
                    persona_prompt,
                    task_directive,
                    "Bundle review brief:\n" + brief,
                    "Bundle generated_at: " + manifest.get("generated_at", ""),
                ]
            ),
        }
    ]

    for scenario in scenarios:
        text_path = bundle_dir / scenario["text_file"]
        png_path = png_dir / pathlib.Path(scenario["svg_file"]).with_suffix(".png").name

        if not text_path.exists():
            raise SystemExit(f"Missing text capture for {scenario['id']}: {text_path}")
        if not png_path.exists():
            raise SystemExit(f"Missing PNG render for {scenario['id']}: {png_path}")

        scenario_header = scenario_template.format_map(
            {
                "id": scenario["id"],
                "title": scenario["title"],
                "width": scenario["width"],
                "height": scenario["height"],
                "focus": scenario["focus"],
                "description": scenario["description"],
            }
        )
        content.append(
            {
                "type": "input_text",
                "text": scenario_header
                + "\n```text\n"
                + text_path.read_text().rstrip("\n")
                + "\n```",
            }
        )
        encoded = base64.b64encode(png_path.read_bytes()).decode("ascii")
        content.append(
            {
                "type": "input_image",
                "image_url": f"data:image/png;base64,{encoded}",
            }
        )

    return content


def call_openai(
    content: list, instructions: str, model: str, schema: dict
) -> dict:
    api_key = os.environ.get("OPENAI_API_KEY", "")
    if not api_key:
        raise SystemExit("OPENAI_API_KEY is not set")

    body = {
        "model": model,
        "instructions": instructions,
        "input": [{"role": "user", "content": content}],
        "text": {
            "format": {
                "type": "json_schema",
                "name": "spotui_tui_review",
                "strict": True,
                "schema": schema,
            }
        },
    }

    req = urllib.request.Request(
        "https://api.openai.com/v1/responses",
        data=json.dumps(body).encode("utf-8"),
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        },
        method="POST",
    )

    try:
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise SystemExit(f"OpenAI request failed: {exc.code} {detail}") from exc


def extract_output_text(response: dict) -> str:
    output_text = (response.get("output_text") or "").strip()
    if not output_text:
        chunks = []
        for item in response.get("output", []):
            for entry in item.get("content", []):
                if entry.get("type") == "output_text" and entry.get("text"):
                    chunks.append(entry["text"])
                if entry.get("type") == "refusal" and entry.get("refusal"):
                    raise SystemExit(f"Model refused review: {entry['refusal']}")
        output_text = "".join(chunks).strip()
    if not output_text:
        raise SystemExit("OpenAI response did not contain JSON output")
    return output_text


def validate_report(report: dict, valid_ids: set) -> int:
    """Validate report fields. Returns blocker count."""
    allowed_verdicts = {"ship", "ship_with_fixes", "needs_another_pass"}
    allowed_confidence = {"high", "medium", "low"}

    if report.get("schema_version") != "1":
        raise SystemExit("schema_version must be '1'")
    if report.get("review_type") != "tui_ui_ux":
        raise SystemExit("review_type must be 'tui_ui_ux'")
    if report.get("persona") != "apple_terminal_ui_ux_designer":
        raise SystemExit("persona must be 'apple_terminal_ui_ux_designer'")
    if report.get("overall_verdict") not in allowed_verdicts:
        raise SystemExit("overall_verdict is invalid")
    if not str(report.get("summary", "")).strip():
        raise SystemExit("summary must not be empty")

    handoff = report.get("implementation_handoff") or {}
    if not str(handoff.get("audience", "")).strip():
        raise SystemExit("implementation_handoff.audience must not be empty")
    if not str(handoff.get("global_objective", "")).strip():
        raise SystemExit("implementation_handoff.global_objective must not be empty")
    if not isinstance(handoff.get("success_definition"), list) or not handoff["success_definition"]:
        raise SystemExit("implementation_handoff.success_definition must be a non-empty array")

    findings = report.get("findings")
    if not isinstance(findings, list):
        raise SystemExit("findings must be an array")

    seen_finding_ids: set = set()
    blocker_count = 0
    for index, finding in enumerate(findings):
        fid = str(finding.get("id", "")).strip()
        if not fid:
            raise SystemExit(f"findings[{index}].id must not be empty")
        if fid in seen_finding_ids:
            raise SystemExit(f"duplicate finding id: {fid}")
        seen_finding_ids.add(fid)
        severity = finding.get("severity")
        if severity not in {"blocker", "polish"}:
            raise SystemExit(f"findings[{index}].severity is invalid")
        if severity == "blocker":
            blocker_count += 1
        for field in ("title", "problem", "rationale", "requested_change", "acceptance_criteria"):
            if not str(finding.get(field, "")).strip():
                raise SystemExit(f"findings[{index}].{field} must not be empty")
        if finding.get("confidence") not in allowed_confidence:
            raise SystemExit(f"findings[{index}].confidence is invalid")
        if not isinstance(finding.get("implementation_notes"), list):
            raise SystemExit(f"findings[{index}].implementation_notes must be an array")
        if not isinstance(finding.get("preserve"), list):
            raise SystemExit(f"findings[{index}].preserve must be an array")
        scenario_ids = finding.get("scenarios")
        if not isinstance(scenario_ids, list) or not scenario_ids:
            raise SystemExit(f"findings[{index}].scenarios must be a non-empty array")
        unknown = [v for v in scenario_ids if v not in valid_ids]
        if unknown:
            raise SystemExit(f"findings[{index}] references unknown scenarios: {unknown}")

    score_entries = report.get("scenario_scores")
    if not isinstance(score_entries, list) or len(score_entries) != len(valid_ids):
        raise SystemExit("scenario_scores must include one entry for each scenario")
    seen_scores: set = set()
    for index, score_entry in enumerate(score_entries):
        scenario_id = score_entry.get("scenario")
        if scenario_id not in valid_ids:
            raise SystemExit(f"scenario_scores[{index}] references unknown scenario: {scenario_id}")
        if scenario_id in seen_scores:
            raise SystemExit(f"duplicate scenario score for: {scenario_id}")
        seen_scores.add(scenario_id)
        score = score_entry.get("score")
        if not isinstance(score, int) or score < 1 or score > 5:
            raise SystemExit(f"scenario_scores[{index}].score must be an integer between 1 and 5")
        if not str(score_entry.get("notes", "")).strip():
            raise SystemExit(f"scenario_scores[{index}].notes must not be empty")

    return blocker_count


def build_summary(report: dict) -> str:
    verdict = report["overall_verdict"]
    ci_status = report["ci_status"]
    prompt_version = report.get("prompt_version", "?")
    findings = report.get("findings", [])
    blockers = [f for f in findings if f["severity"] == "blocker"]
    polish = [f for f in findings if f["severity"] == "polish"]

    lines = [
        f"## spotui TUI review: `{verdict}` (prompt `{prompt_version}`)",
        "",
        report["summary"],
        "",
        f"- CI status: `{ci_status}`",
        f"- Blockers: {len(blockers)}",
        f"- Polish items: {len(polish)}",
        "",
        "### Blockers",
        "",
    ]
    if blockers:
        for finding in blockers:
            lines.append(f"- `{finding['id']}` [{', '.join(finding['scenarios'])}] {finding['title']}")
            lines.append(f"  Requested change: {finding['requested_change']}")
            lines.append(f"  Acceptance criteria: {finding['acceptance_criteria']}")
    else:
        lines.append("- None.")
    lines.extend(["", "### Polish", ""])
    if polish:
        for finding in polish:
            lines.append(f"- `{finding['id']}` [{', '.join(finding['scenarios'])}] {finding['title']}")
            lines.append(f"  Requested change: {finding['requested_change']}")
            lines.append(f"  Acceptance criteria: {finding['acceptance_criteria']}")
    else:
        lines.append("- None.")
    return "\n".join(lines) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser(description="Run TUI review against a bundle")
    parser.add_argument("--bundle", required=True, help="Path to TUI review bundle directory")
    parser.add_argument(
        "--prompt-set",
        default="review/prompts/tui-review-prompt-set.json",
        help="Path to prompt-set manifest JSON",
    )
    parser.add_argument("--out", required=True, help="Output directory for report files")
    parser.add_argument("--model", default=None, help="Override model from prompt set")
    parser.add_argument(
        "--write-summary",
        action="store_true",
        help="Also write review-summary.md and review-comment.md",
    )
    parser.add_argument(
        "--require-rasterizer",
        action="store_true",
        help="Fail immediately if rsvg-convert is not installed",
    )
    args = parser.parse_args()

    if args.require_rasterizer:
        check_rasterizer()
    elif shutil.which("rsvg-convert") is None:
        raise SystemExit(
            "rsvg-convert not found. Install it with:\n"
            "  apt-get install librsvg2-bin  (Debian/Ubuntu)\n"
            "  brew install librsvg          (macOS)"
        )

    bundle_dir = pathlib.Path(args.bundle)
    out_dir = pathlib.Path(args.out)
    out_dir.mkdir(parents=True, exist_ok=True)

    prompt_set_path = pathlib.Path(args.prompt_set)
    prompt_set = load_prompt_set(prompt_set_path)

    model = args.model or os.environ.get("OPENAI_MODEL") or prompt_set["model"]

    persona_prompt = load_text(pathlib.Path(prompt_set["files"]["persona_prompt"]))
    task_directive = load_text(pathlib.Path(prompt_set["files"]["task_directive"]))
    instructions = load_text(pathlib.Path(prompt_set["files"]["instructions"]))
    scenario_template = load_text(pathlib.Path(prompt_set["files"]["scenario_template"]))
    schema = load_json(pathlib.Path(prompt_set["files"]["schema"]))

    prompt_fingerprint = compute_prompt_fingerprint(prompt_set, model)
    prompt_version = prompt_set["version"]

    manifest = load_json(bundle_dir / "manifest.json")
    brief = load_text(bundle_dir / "review-brief.md")

    with tempfile.TemporaryDirectory() as tmp:
        png_dir = pathlib.Path(tmp)
        rasterize_svgs(bundle_dir, png_dir)

        content = build_content(
            persona_prompt,
            task_directive,
            scenario_template,
            brief,
            manifest,
            bundle_dir,
            png_dir,
        )

        print(f"Calling OpenAI ({model}) with {len(manifest.get('scenarios', []))} scenarios...")
        response = call_openai(content, instructions, model, schema)

    output_text = extract_output_text(response)

    try:
        report = json.loads(output_text)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"OpenAI output was not valid JSON: {exc}") from exc

    valid_ids = {s["id"] for s in manifest.get("scenarios", [])}
    blocker_count = validate_report(report, valid_ids)

    verdict = report["overall_verdict"]
    ci_status = "fail" if blocker_count > 0 or verdict == "needs_another_pass" else "pass"

    report["prompt_version"] = prompt_version
    report["prompt_fingerprint"] = prompt_fingerprint
    report["prompt_model"] = model
    report["ci_status"] = ci_status
    report["bundle_generated_at"] = manifest.get("generated_at", "")

    report_path = out_dir / "review-report.json"
    report_path.write_text(json.dumps(report, indent=2) + "\n")
    print(f"Report written to {report_path}")

    if args.write_summary:
        summary = build_summary(report)
        summary_path = out_dir / "review-summary.md"
        summary_path.write_text(summary)

        comment = "\n".join(
            [
                "<!-- spotui-tui-review:start -->",
                summary.rstrip(),
                "",
                "```json",
                json.dumps(report, indent=2),
                "```",
                "<!-- spotui-tui-review:end -->",
                "",
            ]
        )
        comment_path = out_dir / "review-comment.md"
        comment_path.write_text(comment)

        github_output_file = os.environ.get("GITHUB_OUTPUT")
        if github_output_file:
            with open(github_output_file, "a", encoding="utf-8") as fh:
                fh.write(f"ci_status={ci_status}\n")
                fh.write(f"report_path={report_path}\n")
                fh.write(f"summary_path={summary_path}\n")
                fh.write(f"comment_path={comment_path}\n")

    findings = report.get("findings", [])
    blockers = [f for f in findings if f["severity"] == "blocker"]
    polish = [f for f in findings if f["severity"] == "polish"]
    print(
        f"Verdict: {verdict} | CI: {ci_status} | "
        f"Blockers: {len(blockers)} | Polish: {len(polish)} | "
        f"Prompt: {prompt_version} ({prompt_fingerprint})"
    )

    if ci_status != "pass":
        sys.exit(1)


if __name__ == "__main__":
    main()
