CREATE TABLE event_email_templates (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    uuid NOT NULL REFERENCES events(id),
    type        varchar NOT NULL DEFAULT 'invitation',
    config      jsonb NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    updated_by  uuid REFERENCES users(id),
    CONSTRAINT uq_event_email_templates_event_type UNIQUE (event_id, type)
);

CREATE INDEX idx_event_email_templates_event_id ON event_email_templates(event_id);
