# App Samples Tier 3 Additions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 5 Tier 3 samples (nginx-reverse-proxy, ghost-blog, gitea, minio-n8n, ollama-webui) to the existing `conoha-cli-app-samples` repo and update the root README.

**Architecture:** Each sample is a flat top-level directory. Most use official Docker images with compose-only setups (no custom Dockerfile). Japanese READMEs, English code comments/output.

**Tech Stack:** nginx, Ghost, Gitea, MinIO, n8n, Open WebUI, Ollama

**Repo:** `/home/tkim/dev/crowdy/conoha-cli-app-samples`

---

Tier 3 samples are "実務特化" (practical/operational) — they demonstrate real infrastructure patterns rather than framework samples.
