from contextlib import contextmanager
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import publish as publication
from publish import parse_tag, require_checks, validate_inputs


@contextmanager
def repository():
    previous = Path.cwd()
    with tempfile.TemporaryDirectory() as directory:
        root = Path(directory).resolve()
        os.chdir(root)
        try:
            publication.run("git", "init", "-q", "-b", "main")
            publication.run("git", "config", "user.name", "Example Test")
            publication.run("git", "config", "user.email", "example@example.invalid")
            (root / ".github").mkdir()
            (root / ".github/example-channels.json").write_text(json.dumps({
                "schema_version": "spine.example-channels/v1", "cli_version": "v0.4.2",
                "modules": ["."], "application_ci": False,
            }))
            (root / "go.mod").write_text("module example.com/demo\n\ngo 1.25.0\n\nrequire github.com/NARUBROWN/spine v0.6.5\n")
            publication.run("git", "add", ".")
            publication.run("git", "-c", "commit.gpgsign=false", "commit", "-qm", "Initial example")
            yield root, publication.run("git", "rev-parse", "HEAD")
        finally:
            os.chdir(previous)


class FakeGitHub:
    repository = "NARUBROWN/example"

    def __init__(self, sha, existing_sha=None):
        self.sha = sha
        self.existing_sha = existing_sha
        self.writes = []

    def api(self, path, data=None):
        if data is not None:
            self.writes.append((path, data))
            return data
        if path.startswith("git/ref/heads/"):
            return {"object": {"sha": self.sha}}
        if path.startswith("git/matching-refs/tags/"):
            version = path.removeprefix("git/matching-refs/tags/")
            return [{"ref": "refs/tags/" + version, "object": {"type": "commit", "sha": self.existing_sha}}] if self.existing_sha else []
        raise AssertionError(path)


class PublishTests(unittest.TestCase):
    def simulate(self, root, github, branch, version, sha, source=""):
        real_run = publication.run

        def execute(*args):
            if args[0] == "gh":
                return json.dumps([{"check_runs": [{
                    "name": "Example Channel / " + branch, "id": 1, "head_sha": sha,
                    "app": {"id": 15368}, "status": "completed", "conclusion": "success",
                }]}])
            return real_run(*args)

        with patch("publish.run", side_effect=execute):
            return publication.publish(github, root, branch, version, sha, source)

    def test_all_channels_and_stepwise_promotion(self):
        sha = "a" * 40
        self.assertEqual(validate_inputs("preview", "v0.6.6-preview.1", sha, ""), "preview")
        self.assertEqual(validate_inputs("rc", "v0.6.6-rc.1", sha, "v0.6.6-preview.1"), "rc")
        self.assertEqual(validate_inputs("main", "v0.6.6", sha, "v0.6.6-rc.1"), "stable")
        for args in (("main", "v0.6.6", sha, "v0.6.6-preview.1"),
                     ("rc", "v0.6.6-rc.1", sha, "v0.6.5-preview.1"),
                     ("preview", "v0.6.6-preview.1", sha, "v0.6.5"),
                     ("main", "v0.6.6-rc.1", sha, "v0.6.6-preview.1"),
                     ("main", "v0.6.6", "short", "v0.6.6-rc.1")):
            with self.subTest(args=args), self.assertRaises(ValueError):
                validate_inputs(*args)

    def test_malformed_tags_are_rejected(self):
        for tag in ("latest", "main", "v0.6.6-preview.0", "v0.6.6-rc.01", "v00.6.6", "v0.6.6\n"):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                parse_tag(tag)

    def test_release_checks_must_match_current_sha_and_latest_attempt(self):
        sha = "a" * 40
        check = {"name": "Example Channel / main", "head_sha": sha, "id": 1,
                 "app": {"id": 15368}, "status": "completed", "conclusion": "success"}
        require_checks([check], "main", False, sha)
        for mutation in ({"head_sha": "b" * 40}, {"app": {"id": 1}}, {"conclusion": "failure"},
                         {"status": "in_progress"}, {"name": "Example Channel / preview"}):
            with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                require_checks([{**check, **mutation}], "main", False, sha)
        with self.assertRaises(ValueError):
            require_checks([check, {**check, "id": 2, "status": "in_progress"}], "main", False, sha)
        with self.assertRaises(ValueError):
            require_checks([check], "main", True, sha)

    def test_preview_and_rc_publish_tags_but_only_stable_creates_release(self):
        with repository() as (root, sha):
            for branch, version, source in (("preview", "v0.1.0-preview.1", ""),
                                             ("rc", "v0.1.0-rc.1", "v0.1.0-preview.1"),
                                             ("main", "v0.1.0", "v0.1.0-rc.1")):
                if source:
                    publication.run("git", "tag", source, sha)
                github = FakeGitHub(sha)
                report = self.simulate(root, github, branch, version, sha, source)
                self.assertEqual(report["github_release"], branch == "main")
                self.assertEqual([path for path, _ in github.writes],
                                 ["git/refs", "releases"] if branch == "main" else ["git/refs"])

    def test_changed_tree_cannot_be_promoted_even_if_tag_is_ancestor(self):
        with repository() as (root, sha):
            publication.run("git", "tag", "v0.1.0-preview.1", sha)
            (root / "changed.txt").write_text("Unreleased change\n")
            publication.run("git", "add", ".")
            publication.run("git", "-c", "commit.gpgsign=false", "commit", "-qm", "Change example")
            head = publication.run("git", "rev-parse", "HEAD")
            github = FakeGitHub(head)
            with self.assertRaises(ValueError):
                self.simulate(root, github, "rc", "v0.1.0-rc.1", head, "v0.1.0-preview.1")
            self.assertEqual(github.writes, [])

    def test_merge_commit_with_the_same_tree_can_be_promoted(self):
        with repository() as (root, _):
            publication.run("git", "checkout", "-qb", "preview")
            publication.run("git", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-qm", "Preview")
            publication.run("git", "tag", "v0.1.0-preview.1")
            publication.run("git", "checkout", "-q", "main")
            publication.run("git", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-qm", "RC baseline")
            publication.run("git", "-c", "commit.gpgsign=false", "merge", "--no-ff", "preview", "-qm", "Promote Preview")
            sha = publication.run("git", "rev-parse", "HEAD")
            github = FakeGitHub(sha)
            self.simulate(root, github, "rc", "v0.1.0-rc.1", sha, "v0.1.0-preview.1")
            self.assertEqual([path for path, _ in github.writes], ["git/refs"])

    def test_existing_tags_are_never_moved_and_same_sha_can_resume(self):
        with repository() as (root, sha):
            github = FakeGitHub(sha, existing_sha="b" * 40)
            with self.assertRaises(ValueError):
                self.simulate(root, github, "preview", "v0.1.0-preview.1", sha)
            self.assertEqual(github.writes, [])
            github = FakeGitHub(sha, existing_sha=sha)
            self.simulate(root, github, "preview", "v0.1.0-preview.1", sha)
            self.assertEqual(github.writes, [])

    def test_moving_branch_is_rejected_before_any_write(self):
        with repository() as (root, sha):
            github = FakeGitHub("b" * 40)
            with self.assertRaises(ValueError):
                self.simulate(root, github, "preview", "v0.1.0-preview.1", sha)
            self.assertEqual(github.writes, [])


if __name__ == "__main__":
    unittest.main()
