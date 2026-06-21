## High-Level Architecture
The system follows a microservices pattern, strictly separating the heavy, asynchronous workload of document processing from the low-latency, synchronous workload of user querying.

### Core Components
1. **Ingestion API (Go):** A lightweight HTTP server that receives documents and immediately publishes a `DocumentUploaded` event to message broker.
2. **Message Broker (Redpanda):** Acts as the asynchronous buffer, decoupling the upload endpoint from the computationally expensive embedding process.
3. **Embedding Worker (Go):** Consumes messages from Redpanda, parses and chunks the document text, queries the local Ollama instance for embeddings, and upserts the vectors into Qdrant.
4. **Vector Database (Qdrant):** Stores vector embeddings alongside document metadata for fast K-nearest neighbor (KNN) similarity search.
5. **LLM Engine (Ollama):** Hosts the open-source models locally. 
    * *Embedding Model:* `nomic-embed-text`
    * *Generation Model:* `llama3` (or equivalent)
6. **Query Service (Python / FastAPI):** The synchronous backend for user interaction. It orchestrates the retrieval from Qdrant and the prompt construction, then streams the generation from Ollama.
7. **Observability Stack (OpenTelemetry, Prometheus, Grafana):** Captures detailed latency metrics, specifically isolating vector retrieval time vs. token generation time.

## System Design Choices & Trade-offs
* **Custom Python Orchestration vs. LangChain:** Building the orchestration via the official Ollama Python SDK prevents framework lock-in, reduces bloat, and demonstrates a deep understanding of the underlying mechanics (context window management, token streaming, prompt templating).
* **Go for Ingestion vs. Python:** Go's superior concurrency model (Goroutines) makes it ideal for handling parallel chunking and embedding API calls, maximizing throughput during bulk document uploads.
* **Message Broker vs. Direct RPC:** Introducing Message Broker prevents the ingestion service from dropping requests during sudden spikes in document uploads, acting as a shock absorber for the GPU/CPU-bound embedding process.