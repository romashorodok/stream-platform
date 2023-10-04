

-- Protobuf generated type. Values can be found in ingest.pb.go at `IngestEgressType_value'

CREATE TYPE EGRESS AS ENUM ('STREAM_TYPE_HLS', 'STREAM_TYPE_WEBRTC');

CREATE TABLE active_stream_egresses (
       id UUID NOT NULL DEFAULT uuid_generate_v4(),
       active_stream_id UUID NOT NULL,

       type EGRESS NOT NULL,


       FOREIGN KEY (active_stream_id) REFERENCES active_streams(id) ON DELETE CASCADE,
       PRIMARY KEY (id)
);

