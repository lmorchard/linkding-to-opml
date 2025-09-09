# Dev Session Notes: OPML to Linkding

## Session Start
Date: 2025-09-09
Time: 09:21
Branch: opml-to-linkding

## Progress Notes

### Import Command Design Decisions
- Duplicate handling: CLI/config option to update or skip (default: skip)
- Data to import: URL, title, description (from OPML → feed → HTML page, in that priority)
- Tagging: CLI/config option to add custom tags for the entire import run
- Missing HTML pages: Log as INFO and create bookmark with feed URL
- Import strategy: Progressive (import as we go), with detailed logging
- Dry run mode: Available as CLI option to preview without creating bookmarks
- Rate limiting: No delays, no retries, configurable concurrency (default: 16)
- Summary stats: Display at end (imported, skipped, failed counts)
- Log levels: --verbose for INFO, --debug for DEBUG, default shows errors + stats
- OPML structure: Traverse nested folders but flatten to single list (ignore hierarchy)
- Filtering: None for now (import all entries)
- Authentication: Same as export (API token and URL from config/env)

## Final Summary
[To be completed at end of session]