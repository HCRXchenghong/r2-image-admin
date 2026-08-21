# Changelog

## v2.0.0 - 2026-08-21

### Added

- Added the Element Plus based admin UI for dashboard, gallery, AI image generation, settings, and the authenticated R2 setup guide.
- Added richer dashboard telemetry including image totals, storage usage, category distribution, memory, disk, uptime, Go version, goroutines, and GC counters.
- Added save-and-auto-restart configuration flow so setting changes can be written and applied without manual process handling.
- Added Cloudflare R2 storage support through the S3 API, including presigned upload APIs and an authenticated guide page for finding and filling R2 fields.
- Added direct original-image upload for generated or pre-optimized raster images.

### Security

- Hardened JWT validation, response security headers, exact-origin CORS, request body limits, upload validation, login rate limiting, and audit logging.
- Added production configuration checks for strong admin passwords, strong JWT secrets, HTTPS public URLs, and non-local production storage.
- Blocked SVG uploads and private/local AI gateway targets.

### Notes

- Real Cloudflare R2 availability still depends on valid R2 credentials, bucket permissions, CORS, and a working public/custom domain in the deployment environment.
- Full Level 2 MLPS compliance also requires deployment and operations evidence such as HTTPS reverse proxy, host baseline, database backup/TLS, centralized logging, and account governance.
