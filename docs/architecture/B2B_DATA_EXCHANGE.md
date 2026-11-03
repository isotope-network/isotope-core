# ISOTOPE Enterprise: B2B Data Exchange Without Disclosing Identity

## Status: concept

ISOTOPE Enterprise is a commercial extension for data exchange between organizations (banks, insurers, medical institutions) without disclosing personal data.

Not implemented. Part of the roadmap.

---

## The Problem

Today, organizations exchange data through centralized bureaus (credit histories, medical registries). This leads to:

- **Excessive disclosure** — you receive more data than needed for an answer
- **Loss of control** — the client does not know who requested what about them
- **Single point of failure** — a centralized database can be hacked or forced to disclose everything

---

## The ISOTOPE Solution

Instead of «tell everything about the client» — «answer a specific question: yes or no».

### Example: Credit Scoring

**Today:** A bank requests a credit history → receives EVERYTHING.

**With ISOTOPE:**

1. The client generates an anonymous ID and gives the bank permission to query.
2. The bank asks: «Number of delinquencies > 3 in 2 years?» → receives «no».
3. The bank asks: «Current debt burden > 50%?» → receives «no».
4. No bank discloses its clients' data — only answers to questions.

### Example: Medical Underwriting

**Today:** An insurer requests a medical history → receives diagnoses.

**With ISOTOPE:**

1. The hospital calculates a risk coefficient (a number, not a diagnosis).
2. The insurer asks: «Cardiovascular risk coefficient?» → receives «0.88».
3. The insurer receives a number for tariff calculation without knowing diagnoses.

---

## How It Works Technically

1. **Anonymization:** The client receives an ID = hash(data + salt). A meaningless string for everyone except their own bank.
2. **Query to the network:** The gossip protocol broadcasts the query across nodes.
3. **Answer without disclosure:** The node that knows this ID answers only the specific question.
4. **Weighting of answers:** The ethical engine aggregates answers taking into account node weights.

---

## Tokenomics in B2B

- The bank pays tokens for queries.
- The node receives tokens for useful answers.
- Quality answers (from nodes with high weight) are valued higher.
- Deception is punished by a drop in weight.

---

## Legal Compatibility

| Law | How ISOTOPE Complies |
|-----|----------------------|
| **GDPR** | Right to be forgotten (data decay). Data minimization (only answers). Client consent. |
| **152-FZ** | Anonymized data. Banking secrecy (answers are not personal data). |
| **HIPAA** | Minimum necessary information. De-identification. Access audit. |

---

## Enterprise Roadmap

| Stage | What |
|-------|------|
| **E0** | Consortium formation (participants, agreement, memorandum) |
| E1 | Private subnets |
| E2 | Differentiated access rights |
| E3 | Audit trail for the client |
| E4 | Zero-Knowledge Proofs (optional) |
| E5 | Standard contract + certification |
| E6 | Pilot: 3 banks + 1 insurer |

---

**ISOTOPE Enterprise is not the sale of data. It is infrastructure for exchanging answers without disclosure.**