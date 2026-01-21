# Aura Project Technical Specification

This document describes the Aura WhatsApp-over-DNS tunnel architecture, data flow, and operational guidance. It is formatted as a GitHub-style README.

## Architecture Overview

Aura tunnels TCP payloads through DNS AAAA queries and responses. The client fragments upstream TCP data into small chunks, encodes them into DNS labels, and sends them to a DNS server. The server decodes chunks, reassembles them in sequence, and forwards the data to the target (e.g., WhatsApp). Downstream traffic is buffered by the server and returned to the client on polling queries.

### Mermaid Sequence Diagram

```mermaid
sequenceDiagram
    participant Client
    participant DNS
    participant Server

    Client->>DNS: AAAA Query (nonce-seq-session.data.domain)
    DNS->>Server: DNS request
    Server->>Server: Decode + buffer + reassemble
    Server->>Server: Forward reassembled bytes to target
    Server-->>DNS: AAAA Answer (downstream bytes)
    DNS-->>Client: AAAA Answer (downstream bytes)

    Note over Client,Server: Client repeats with seq increments; polls with seq=ffff for downstream data.
```

## Detailed Logic Flow: Sequence Reassembly

The server maintains per-session state to reconstruct an ordered byte stream from DNS chunks. The reassembly logic follows these steps:

1. **Decode and parse:**
   - Parse query fields (nonce, seq, session ID, data label).
   - Decode Base32 label into raw bytes.
2. **Session lookup:**
   - Retrieve or create the session and its TCP connection to the target.
3. **Initial sequence control:**
   - The server expects sequence `0000` as the start of the stream.
   - If `expectedSeq == 0` and a higher seq arrives, the server starts an **InitialTimeout** timer.
   - If `seq 0000` does not arrive before the timer expires, the session is closed to prevent unbounded buffering.
4. **Queue chunk:**
   - Store the chunk in `pendingChunks[seq]`. If a chunk for the same seq already exists, append (handles duplicate labels).
5. **Process in-order chunks:**
   - While `pendingChunks` contains `expectedSeq`:
     - Pop that chunk.
     - If still in TLS handshake mode, append to the handshake buffer and only flush once the full TLS record length is met.
     - Otherwise, write the chunk directly to the target TCP socket.
     - Increment `expectedSeq` and continue until the next expected seq is missing.

This logic ensures that out-of-order packets are buffered until the missing sequence arrives, while preventing indefinite memory growth if the initial packet never arrives.

## Session State Machine

| State | Entry Condition | Actions | Exit Condition |
| --- | --- | --- | --- |
| **Init** | Session created | Set `expectedSeq=0`, start reader goroutine | First upstream data arrives |
| **Handshaking** | First in-order data begins with TLS ClientHello (0x16) | Buffer TLS record until length is satisfied | Full TLS record flushed to target |
| **Streaming** | TLS record completed or non-TLS first byte | Forward in-order chunks directly | Session timeout or connection error |
| **Timeout** | Session idle beyond `SessionTimeout` or initial seq timeout | Close TCP conn, drop session | Session removed |

## Troubleshooting Guide

### Packet Loss (Missing seq 0000)
**Symptom:** Connection stalls, DNS queries continue but no progress.  
**Cause:** First packet never arrives; server holds buffered chunks.  
**Mitigation:** The server uses `InitialTimeout` to close stalled sessions so the client can retry. Ensure upstream packet delivery or reduce MTU to lower fragmentation.

### Out-of-Order Delivery
**Symptom:** Slow handshake or delayed forwarding.  
**Cause:** DNS queries may arrive out of order.  
**Mitigation:** Server buffers chunks by sequence and only forwards when in order. Excessive reordering may increase latency. Consider reducing DNS resolver recursion hops.

### Latency Spikes on Downstream
**Symptom:** Message delays or bursty delivery.  
**Cause:** Downstream data is returned only during poll intervals.  
**Mitigation:** Decrease the poll interval (client) or increase response packing limits. Ensure resolver response time is stable.

### DNS Response Truncation
**Symptom:** Partial downstream data or slow throughput.  
**Cause:** DNS response size limits or resolver behavior.  
**Mitigation:** Limit AAAA records per response and keep payloads small. Avoid EDNS restrictions.

### Session Timeouts
**Symptom:** Tunnel drops during idle periods.  
**Cause:** `SessionTimeout` expires when no DNS activity occurs.  
**Mitigation:** Client should keep polling even when idle to refresh `lastSeen`.
