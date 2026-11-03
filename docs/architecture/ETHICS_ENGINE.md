# ISOTOPE Ethics Engine

## Overview

The ethics engine is a neural network that evaluates every message against the network's ethical hash.

This is not censorship. This is an immune system.

ISOTOPE ethics already works with messages. In the future (v3.0+), it will also check AI models: before loading onto a node, a model will receive an ethical passport — a hash of its architecture and training data. This is needed to prevent a harmful model from entering the network. More details in the ISOTOPE AI Mesh concept.

---

## Ethical Hash

### What It Is

The ethical hash is the digital DNA of the network. A string transformed into a 100-dimensional vector via SHA-256.

The current hash is based on seven universal commandments:

- Do no harm.
- Do not steal.
- Do not deceive.
- Do not betray.
- Do not be indifferent.
- Create, do not destroy.
- Treat others as you would yourself.

These principles are gathered from all the great teachings of humanity. They belong to no single religion, no single ideology.

### How It Works

1. The string of commandments is hashed into SHA-256 → `hashText()`.
2. The hash is transformed into a 100-dimensional vector → `hashToVector()`.
3. The reference vector is compared with each message's vector → `cosineSimilarity()`.
4. Initial message weight = `0.3 + similarity × 0.5`.

---

## ISOTOPE Ethical Space

### Cross-Network Compatibility

Different language networks (ISOTOPE-RU, ISOTOPE-EN, ISOTOPE-DE) are connected not by translation bridges, but by ethical space.

Each network publishes its ethical vector — a semantic projection of the seven commandments. Networks compare vectors via cosine similarity.

Compatibility threshold: **> 0.7** (adaptively raised).

Compatibility does not mean translation. It means trust. Messages between networks pass only if their vectors are close.

### How It Works Technically

1. Network A computes the vector of its ethical hash.
2. Network B computes the vector of its ethical hash.
3. Nodes compare vectors via `cosineSimilarity()`.
4. If similarity > 0.7 — networks are compatible, exchange is allowed.
5. If not — connection is not established.

Translation (if needed) is performed locally on the recipient's device. Vectors solve the task of trust, not translation.

---

## Neural Network

### Architecture

- **Dimension:** 100 neurons per layer (VectorDim = 100).
- **Layers:** grow every 20 messages.
- **Activation:** clamp from -1 to 1.
- **Training:** gradient descent on the last layer.

### Forward
inputVector (100 numbers) → layer 1 → layer 2 → ... → outputVector

text

Each layer multiplies the vector by its weights and clamps the values.

### Train
error = target - output
weights += error × learningRate
learningRate = 0.01 × message_weight

text

---

## Text Vectorization

### textToVector

Transforms text into a 100-dimensional vector:

1. **30%** — individual characters: letter codes are distributed across 100 dimensions.
2. **70%** — bigrams: character pairs are hashed into SHA-256 and distributed.
3. **Normalization** — division by maximum.

### Why Bigrams

«kill» → `ki`, `il`, `ll`.

«help» → `he`, `el`, `lp`.

Different bigrams → different vectors → the network distinguishes meaning.

---

## Collective Learning

### Like

- Message weight: +0.15.
- Neural network training: `train(input, output, ethicsVec, lr)` — pulls toward the reference.
- At 5 likes: weight = 1.0, training intensified.

### Dislike

- Message weight: -0.15.
- Neural network training: `train(input, output, invertedInput, lr)` — pushes away from the reference.
- At 4 dislikes: weight < 0.15 → archiving.

---

## Personal Filters

### preHash

The user enters words they want to see. The preHash vector attracts messages with similar meaning.

Formula: `weight = ethicsScore × 0.4 + preSimilarity × 0.6`.

### antiHash

The user enters words they do not want to see. The antiHash vector repels unwanted messages.

Formula: `weight = weight × (1 - antiSimilarity × 0.8)`.

---

## Message Lifecycle

New message → Ethical similarity (hash) → preHash (attraction) → antiHash (repulsion) → Final weight → Likes/dislikes (training) → Aging (-0.01/hour) → Weight < 0.15 → archiving → Weight < 0.1 → deletion.

---

## Limitations

- **100 dimensions** — good for short phrases, but not for semantic analysis.
- **Bigrams** — distinguish words, but not context.
- **Collective learning** — requires thousands of ratings for a noticeable effect.
- **preHash/antiHash** — conflict on short words.

---

## Future

- Increasing dimension (100 → 300 → 1000).
- Transformers instead of simple layers.
- Federated learning between nodes.
- Distributed AI inference (ISOTOPE AI Mesh).
- Ethical passport of a model (for AI Mesh).
- Ethical space between language networks.