# Project Issues & Missing Resources Tracker

This document tracks known issues with the school-provided subject materials and our applied fixes.

---

## 1. Missing Subject Attachments (Not Provided by Staff)

After a full review of the `tree.pdf` subject requirements and the intranet project page, it appears the pedagogy team forgot to upload the required test binaries.

**The Test Binaries Archive**
On **Page 22** (Chapter VIII.6: Test Scenario) of the subject, it states:
> *"An archive containing two helper binaries is provided with this subject. These tools can be used to validate your implementation and will also be used during evaluation."*

This archive is supposed to contain:
- **`ft_aquarium`**: A visualizer program that connects to the server and renders messages as fish.
- **`ft_fish`**: A producer application that spam-publishes random fish messages.

**Status:** MISSING. Not provided on the intranet. 
**Workaround:** We will use standard POSIX Named Pipes and basic Linux commands (`echo` and `cat`) to test the server's queues until the staff uploads the binaries.

---

## 2. Gitinette Evaluation Script Bug

**The Symptom**
The provided `gitinette` script throws a false warning:
> `Warnings (manual review needed): X commit(s) landed directly on main outside of a merge commit...`

**The Cause**
The `list_commits` function runs `git log main`, which grabs the entire repository history (including the hidden commits inside merged feature branches). Then, `check_merge_strategy` flags any commit with exactly `1` parent. However, normal commits made *inside* a feature branch also have exactly 1 parent. It incorrectly flags perfectly legal feature branch commits as direct commits to main.

**The Fix**
We rewrote the `gitinette` script locally:
1. `list_commits` now returns *everything* (so the commit message checker and the random `--evaluation` sampler have the full pool of your work).
2. Inside `check_merge_strategy`, it now runs a *second*, completely isolated git command (`git log --first-parent`) just to check for the direct-to-main violations.

The script now successfully passes without false warnings and correctly samples 3 commits for `--evaluation`.
