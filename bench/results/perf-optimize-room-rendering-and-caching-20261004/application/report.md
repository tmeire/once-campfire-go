# Go and Rust application benchmark

Native release binaries; direct HTTP/1.1 application listeners; identity encoding; identical seed and CPU affinity. Media use installed native libraries. Workstation background load recorded.

3 alternating repetitions; 5-second HTTP samples after 2-second warmups. Four application workers on CPUs 0,1,2,3; load generator on CPUs 4,5,6,7. HTTP uses identity encoding. Every run starts with a separate copy of the same seed.

HTTP response contracts compare message IDs, sidebar room IDs, avatar bytes and CSS bytes. Write checks require every successful request to persist and enter the FTS index. Cable checks require every client to subscribe and every message to arrive at every client. Upload checks fetch the actual representation and compare its bytes across applications.

## HTTP

Rates show median (minimum–maximum); latency is the median of each run’s percentile.

| Workload | Clients | App | Requests/s (range) | p50 ms | p90 ms | p99 ms |
|---|---:|---|---:|---:|---:|---:|
| room_show | 1 | go-before | 2,075 (1,309–2,160) | 0.308 | 0.610 | 3.865 |
| room_show | 1 | go | 3,540 (3,464–3,728) | 0.213 | 0.317 | 1.547 |
| room_show | 16 | go-before | 5,242 (3,093–5,533) | 2.133 | 6.639 | 12.975 |
| room_show | 16 | go | 7,372 (7,339–9,731) | 1.398 | 4.059 | 10.999 |
| room_show | 64 | go-before | 4,797 (4,713–5,137) | 11.623 | 21.103 | 48.127 |
| room_show | 64 | go | 9,313 (9,192–9,394) | 6.111 | 11.735 | 18.959 |
| messages_page | 1 | go-before | 3,106 (3,088–3,133) | 0.229 | 0.360 | 2.699 |
| messages_page | 1 | go | 4,207 (4,206–4,316) | 0.190 | 0.256 | 1.085 |
| messages_page | 16 | go-before | 8,040 (7,858–8,321) | 1.358 | 4.367 | 9.311 |
| messages_page | 16 | go | 11,017 (10,778–11,272) | 1.007 | 2.929 | 6.835 |
| messages_page | 64 | go-before | 8,036 (7,143–8,148) | 7.271 | 13.119 | 20.479 |
| messages_page | 64 | go | 10,983 (10,960–11,205) | 5.067 | 10.255 | 17.407 |
| sidebar | 1 | go-before | 3,464 (3,452–3,538) | 0.221 | 0.305 | 2.129 |
| sidebar | 1 | go | 3,894 (3,839–3,914) | 0.204 | 0.274 | 1.628 |
| sidebar | 16 | go-before | 9,857 (9,507–9,873) | 1.265 | 3.051 | 6.527 |
| sidebar | 16 | go | 10,860 (10,827–11,099) | 1.130 | 2.801 | 6.187 |
| sidebar | 64 | go-before | 9,174 (8,975–9,692) | 6.339 | 11.407 | 16.623 |
| sidebar | 64 | go | 10,184 (9,987–10,198) | 5.707 | 10.551 | 15.279 |
| search | 1 | go-before | 2,390 (2,344–2,411) | 0.301 | 0.467 | 3.431 |
| search | 1 | go | 2,501 (2,475–2,626) | 0.286 | 0.448 | 3.295 |
| search | 16 | go-before | 6,839 (6,675–7,001) | 1.737 | 4.819 | 9.775 |
| search | 16 | go | 7,360 (7,293–7,555) | 1.515 | 4.583 | 9.607 |
| search | 64 | go-before | 6,318 (6,036–6,363) | 9.551 | 15.583 | 23.775 |
| search | 64 | go | 6,830 (6,553–6,903) | 8.735 | 15.047 | 23.183 |
| avatar | 1 | go-before | 3,896 (3,866–4,074) | 0.208 | 0.270 | 1.251 |
| avatar | 1 | go | 4,178 (4,164–4,205) | 0.199 | 0.254 | 1.023 |
| avatar | 16 | go-before | 15,081 (14,632–15,232) | 0.753 | 2.063 | 5.811 |
| avatar | 16 | go | 16,462 (16,350–16,669) | 0.671 | 1.884 | 5.283 |
| avatar | 64 | go-before | 14,208 (13,893–14,463) | 3.911 | 7.931 | 13.087 |
| avatar | 64 | go | 15,868 (15,759–16,051) | 3.413 | 7.275 | 12.791 |
| static_css | 1 | go-before | 15,369 (15,076–15,655) | 0.056 | 0.071 | 0.164 |
| static_css | 1 | go | 15,273 (15,207–15,326) | 0.057 | 0.072 | 0.169 |
| static_css | 16 | go-before | 111,144 (107,858–112,109) | 0.087 | 0.226 | 1.392 |
| static_css | 16 | go | 109,008 (107,112–113,394) | 0.089 | 0.229 | 1.348 |
| static_css | 64 | go-before | 117,815 (116,325–120,713) | 0.306 | 1.032 | 4.523 |
| static_css | 64 | go | 115,718 (115,430–118,361) | 0.313 | 1.019 | 4.667 |
| post_message | 1 | go-before | 1,144 (1,100–1,147) | 0.699 | 1.109 | 4.275 |
| post_message | 1 | go | 1,169 (1,153–1,176) | 0.686 | 1.038 | 4.435 |
| post_message | 16 | go-before | 1,621 (1,509–1,632) | 7.263 | 21.311 | 42.623 |
| post_message | 16 | go | 1,648 (1,640–1,658) | 7.183 | 20.799 | 40.351 |
| post_message | 64 | go-before | 1,579 (1,498–1,581) | 28.975 | 91.647 | 181.119 |
| post_message | 64 | go | 1,593 (1,579–1,628) | 28.191 | 90.431 | 178.303 |

## Action Cable

Throughput counts posted messages delivered to **all** clients; one such message produces one frame per client. Latency includes delivery, rather than just accepting a post. The reference load generator reports p90 rather than p95.

| Clients | Deflate | App | Messages/s (range) | Frames/s | Per-client p99 ms | All-clients p99 ms | Wire MB/s |
|---:|---|---|---:|---:|---:|---:|---:|
| 100 | False | go-before | 874.5 (870.9–880.5) | 87,446 | 14.343 | 17.055 | not measured |
| 100 | False | go | 878.2 (869.2–903.0) | 87,821 | 14.351 | 16.655 | not measured |
| 100 | True | go-before | 759.8 (713.8–785.8) | 75,980 | 17.775 | 22.623 | 202.1 |
| 100 | True | go | 754.5 (723.5–775.9) | 75,455 | 17.887 | 21.215 | 200.7 |
| 1,000 | False | go-before | 149.4 (148.6–151.6) | 149,372 | 82.303 | 148.735 | not measured |
| 1,000 | False | go | 153.3 (148.9–154.9) | 153,322 | 76.543 | 118.591 | not measured |
| 1,000 | True | go-before | 106.7 (101.7–107.5) | 106,668 | 112.383 | 216.575 | 283.8 |
| 1,000 | True | go | 111.2 (107.2–112.7) | 111,177 | 111.231 | 176.511 | 295.9 |
| 10,000 | False | go-before | 10.1 (5.9–10.4) | 100,998 | 438.527 | 667.647 | not measured |
| 10,000 | False | go | 9.8 (8.8–10.7) | 97,855 | 427.775 | 623.615 | not measured |
| 10,000 | True | go-before | 8.8 (7.8–9.1) | 87,568 | 823.295 | 1165.311 | 232.7 |
| 10,000 | True | go | 8.5 (6.4–8.6) | 85,110 | 1046.015 | 1302.527 | 226.1 |

## Media and resources

| App | Upload + thumbnail median ms | Thumbnail bytes | Startup median ms | Idle Pss MiB | After HTTP Pss MiB | Final Pss MiB | Binary MiB |
|---|---:|---:|---:|---:|---:|---:|---:|
| go-before | 88.44 | 84472 | 117.3 | 37.5 | 123.2 | 1190.4 | 36.2 |
| go | 81.41 | 84472 | 90.2 | 38.6 | 135.1 | 1112.7 | 36.2 |

Startup includes application initialization, measured to a successful health request at 25 ms polling intervals. Final memory follows the complete workload sequence and includes allocator high-water effects; it is not a per-client memory measurement. Each media sample creates a new blob and fetches its generated thumbnail.

## Reproduction and limits

- [Raw samples](raw.json) include statuses, errors, latency distributions, delivery counters, byte hashes and memory snapshots. [Metadata](metadata.json) records source/binary/seed hashes, CPU details, affinity and load averages.
- These are local workstation measurements, sequential within the harness. Background host activity is recorded, not eliminated. They are not a language-wide performance claim.
- This comparison uses direct application HTTP listeners. TLS, ACME, gzip/zstd and the public response cache have separate functional tests; these numbers do not measure their throughput.
- Screen-level HTML and network comparisons remain stricter than the functional response contracts used here. Benchmark validation is not a declaration of complete byte-for-byte UI parity.

## Validation totals

6 completed application runs; 149,304 acknowledged HTTP writes verified in both messages and FTS; 30 real thumbnail samples with identical hashes. HTTP errors: 0. Incomplete throughput deliveries: 0.

Generated HTML has different whitespace, attributes and serialization. Representative body sizes from the first repetition are shown below; comparison contracts use matching message/room IDs and the separately verified browser workflows.

| Response | App | Bytes |
|---|---|---:|
| room_show | go-before | 374,036 |
| messages_page | go-before | 342,444 |
| sidebar | go-before | 9,462 |
| search | go-before | 135,497 |
| avatar | go-before | 3,364 |
| static_css | go-before | 1,218 |
| room_show | go | 374,036 |
| messages_page | go | 342,444 |
| sidebar | go | 9,462 |
| search | go | 135,497 |
| avatar | go | 3,364 |
| static_css | go | 1,218 |

Default application logging is retained: Rust logs completed jobs; Go logs startup and job failures. Native media libraries are identical between the two applications in this run. Container byte goldens use the separately pinned toolchain.
