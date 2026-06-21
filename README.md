# Aura
A localized, event-driven Retrieval-Augmented Generation (RAG) system

## Introduction
It bridges high-throughput distributed systems engineering with modern generative AI patterns. 
The system is designed to ingest documents asynchronously, vectorize text using local embedding models, and answer user queries using a locally hosted Large Language Model (LLM). 

**Goals:**
* Build a scalable, decoupled ingestion pipeline using event-driven architecture.
* Implement custom orchestration for LLM inference (avoiding heavy abstractions like LangChain/LlamaIndex).
* Maintain a strict zero-cost, open-source infrastructure running entirely locally or within a standard Kubernetes cluster.
* Instrument the pipeline with comprehensive distributed tracing for AI-specific observability.