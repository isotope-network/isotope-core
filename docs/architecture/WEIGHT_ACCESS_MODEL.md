# ISOTOPE Weight Access Model

## Principle

Node weight is a universal pass.
Not a role. Not a passport. Not a certificate.
Just reputation earned over time
and through useful actions.

---

## Rights Table

| Weight | Rights | Examples |
|--------|--------|----------|
| 0–0.3 | Read (teaser) | Sees 30% of channel content |
| 0.3–0.5 | Full access | Sees everything, writes messages |
| 0.5–0.7 | Comment, moderate | Dislikes spam, writes in discussions |
| 0.7–1.0 | Vote, relay, replicas | Routes Sphinx/Onion, stores replicas |

---

## Where It Applies

| Direction | Application | Status |
|-----------|-------------|--------|
| Messenger | Channels, groups, comments | Planned |
| Enterprise | Bank with weight > 0.7 requests scoring | Planned (v3.0+) |
| AI Mesh | Node with weight > 0.7 runs a model | Planned (v3.0) |
| Governance | Voting on the ethical hash | Planned |
| Moderation | Dislikes only from weight > 0.5 | Planned |
| Replication | Storing replicas only weight > 0.5 | Planned |
| Relay | Sphinx/Onion routing only weight > 0.7 | Sphinx (v2.0.1) — implemented. Onion (full) — planned |

---

## Why This Works

- A spammer has weight 0 → cannot influence
- A bot will not earn weight → will not get rights
- A bank earned weight through honest answers → got access
- A user earned weight through useful messages → got a vote

---

## Anti-Abuse

| Attack | Protection |
|--------|------------|
| Spam with zero weight | Messages fade |
| Weight farming | Months of honest activity are required |
| Bot network | Messages are meaningless, weight does not grow |
| Purchased weight | Weight is bound to PeerID, not for sale |

---

## Emergent Trust

### Principle

Trust is not proven. Trust is earned.

Not «prove you are human.»
But «be useful — and the network will trust you.»

### How It Works

- A message receives weight from the ethical hash
- Likes raise weight, dislikes lower it
- Node weight = average weight of its messages
- High weight = high trust

### What a Bot Cannot Do

- Gain high weight
  (its messages are meaningless and fade)
- Become a relay peer (weight > 0.7 required)
- Influence votings (weight required)
- Pass the ethical passport of a model (for AI Mesh, planned)

### Why «Emergent»

From three simple rules
(message weight, likes/dislikes, average node weight)
reputation arises by itself.
No one issues it. It is emergent.

### Analogy

The immune system does not ask a cell for a passport.
It recognizes «self vs. foreign»
by accumulated experience.