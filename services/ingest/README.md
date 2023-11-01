
### Description
The service use WebRTC as input and output protocol. Additionally, it supports HLS as output protocol

### Ingest architecture

Ingest has implemented a single ingress type. I keep in mind that each lower-level component shouldn't depend on stream type.
By the design egresses must works with different ingress.

![platform](./../../docs/diagram-ingest.jpg)
