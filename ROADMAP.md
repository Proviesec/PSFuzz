# Roadmap

Planned items for future releases. For a broader list of **ideas** (reporting, UX, integrations, polish), see **[IDEAS.md](IDEAS.md)**—north-star and “extra class” features that could be picked up over time. For **architecture** refactors (config split, Engine phases, module config, internal/scan), see the “Architecture (future improvements)” section in [IDEAS.md](IDEAS.md).

## Response modules (planned)

No new modules currently planned. New modules follow the same pattern as existing ones; see [MODULES.md](MODULES.md#adding-a-new-module-for-developers).

## Features (planned)

| Feature | Description |
|---------|-------------|
| **Multipart file upload fuzzing** | Fuzz file upload endpoints: multipart/form-data with file from wordlist (e.g. paths or extensions). Enables testing upload restrictions, extension bypass, LFI via filename. |
