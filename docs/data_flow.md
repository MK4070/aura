## Data Flows

### Asynchronous Document Ingestion Flow
1. **Client** sends a `POST /upload` request with a document to the **Ingestion API**.
2. **Ingestion API** validates the payload, generates a unique `doc_id`, publishes an event to the **Redpanda** `document-events` topic, and returns a `202 Accepted` to the client.
3. **Embedding Worker (Go)** pulls the event from **Redpanda**.
4. Worker chunks the text.
5. Worker makes parallel REST/RPC calls to **Ollama** to vectorize chunks.
6. Worker batch-inserts the vectors and metadata into **Qdrant**.

### Synchronous RAG Query Flow
1. **Client** sends a `POST /query` request to the **Query Service**.
2. **Query Service** calls **Ollama** to embed the user's query string.
3. **Query Service** queries **Qdrant** using the query vector to retrieve the top-K most relevant document chunks.
4. *(Extensibility Point)*: Future re-ranking logic will be inserted here.
5. **Query Service** constructs the final LLM prompt combining the retrieved context and the user query.
6. **Query Service** initiates a streaming inference request to **Ollama**.
7. **Query Service** streams the generated tokens back to the **Client** using Server-Sent Events (SSE).