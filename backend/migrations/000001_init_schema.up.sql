CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================
-- MASTER TABLES
-- ============================================================

CREATE TABLE users (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        varchar NOT NULL,
    email       varchar NOT NULL,
    status      varchar NOT NULL DEFAULT 'active'
                CHECK (status IN ('pending', 'active', 'disabled')),
    password    varchar NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz,
    CONSTRAINT uq_users_email UNIQUE (email)
);

CREATE TABLE roles (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        varchar NOT NULL,
    description varchar,
    created_by  uuid REFERENCES users(id),
    updated_by  uuid REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz,
    CONSTRAINT uq_roles_name UNIQUE (name)
);

-- ============================================================
-- ORGANIZATIONS / MEMBERSHIP
-- ============================================================

CREATE TABLE organizations (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        varchar NOT NULL,
    status      varchar NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'in_review', 'disabled')),
    owner_id    uuid NOT NULL REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz
);

CREATE INDEX idx_organizations_owner_id ON organizations(owner_id);

CREATE TABLE user_roles (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users(id),
    role_id         uuid NOT NULL REFERENCES roles(id),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_user_role_org UNIQUE (user_id, role_id, organization_id)
);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_org_id ON user_roles(organization_id);

-- ============================================================
-- EVENTS
-- ============================================================

CREATE TABLE events (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name            varchar NOT NULL,
    status          varchar NOT NULL DEFAULT 'active',
    description     varchar,
    start_date      date NOT NULL,
    end_date        date,
    start_time      time,
    end_time        time,
    creator_id      uuid NOT NULL REFERENCES users(id),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz,
    CONSTRAINT chk_event_dates CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_events_creator_id ON events(creator_id);
CREATE INDEX idx_events_organization_id ON events(organization_id);

CREATE TABLE seat_categories (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        varchar NOT NULL,
    code        varchar,
    price       numeric(12, 2) NOT NULL DEFAULT 0,
    currency    varchar(3) NOT NULL DEFAULT 'IDR',
    event_id    uuid NOT NULL REFERENCES events(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz,
    CONSTRAINT uq_seat_category_event_code UNIQUE (event_id, code)
);

CREATE INDEX idx_seat_categories_event_id ON seat_categories(event_id);

CREATE TABLE seats (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code        varchar NOT NULL,
    section     varchar,
    "row"       int,
    col         int,
    status      varchar NOT NULL DEFAULT 'available'
                CHECK (status IN ('available', 'reserved', 'occupied', 'blocked')),
    description varchar,
    event_id    uuid NOT NULL REFERENCES events(id),
    category_id uuid NOT NULL REFERENCES seat_categories(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz,
    CONSTRAINT uq_seats_event_code UNIQUE (event_id, code)
);

CREATE INDEX idx_seats_event_id ON seats(event_id);
CREATE INDEX idx_seats_category_id ON seats(category_id);

-- ============================================================
-- GUESTS
-- ============================================================

CREATE TABLE guests (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id      uuid NOT NULL REFERENCES events(id),
    category_id   uuid NOT NULL REFERENCES seat_categories(id),
    name          varchar NOT NULL,
    email         varchar NOT NULL,
    paid_date     timestamptz,
    ticket_count  integer NOT NULL DEFAULT 1 CHECK (ticket_count > 0),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);

CREATE INDEX idx_guests_event_id ON guests(event_id);
CREATE INDEX idx_guests_category_id ON guests(category_id);

-- ============================================================
-- SEAT BOOKINGS (merged guest_seats + onsite_reservations)
-- ============================================================

CREATE TABLE seat_bookings (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    guest_id    uuid NOT NULL REFERENCES guests(id),
    event_id    uuid NOT NULL REFERENCES events(id),
    category_id uuid NOT NULL REFERENCES seat_categories(id),
    seat_id     uuid NOT NULL REFERENCES seats(id),
    source      varchar NOT NULL DEFAULT 'invited'
                CHECK (source IN ('invited', 'onsite')),
    status      varchar NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'not_paid', 'paid', 'cancelled')),
    notes       varchar,
    paid_at     timestamptz,
    created_by  uuid NOT NULL REFERENCES users(id),
    updated_by  uuid REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz
);

CREATE INDEX idx_seat_bookings_guest_id ON seat_bookings(guest_id);
CREATE INDEX idx_seat_bookings_event_id ON seat_bookings(event_id);
CREATE INDEX idx_seat_bookings_seat_id ON seat_bookings(seat_id);

CREATE UNIQUE INDEX uq_seat_bookings_active_seat
    ON seat_bookings(seat_id)
    WHERE deleted_at IS NULL AND status <> 'cancelled';

-- ============================================================
-- PAYMENTS
-- ============================================================

CREATE TABLE payments (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id   uuid NOT NULL REFERENCES seat_bookings(id),
    amount       numeric(12, 2) NOT NULL,
    currency     varchar(3) NOT NULL DEFAULT 'IDR',
    method       varchar,
    gateway_ref  varchar,
    status       varchar NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending', 'success', 'failed', 'refunded')),
    paid_at      timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_payments_booking_id ON payments(booking_id);

-- ============================================================
-- ATTENDANCE
-- ============================================================

CREATE TABLE attendance_logs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    guest_id    uuid NOT NULL REFERENCES guests(id),
    event_id    uuid NOT NULL REFERENCES events(id),
    seat_id     uuid NOT NULL REFERENCES seats(id),
    status      varchar NOT NULL DEFAULT 'not_checked_in'
                CHECK (status IN ('checked_in', 'not_checked_in')),
    message     varchar,
    created_by  uuid NOT NULL REFERENCES users(id),
    updated_by  uuid REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz
);

CREATE INDEX idx_attendance_logs_guest_id ON attendance_logs(guest_id);
CREATE INDEX idx_attendance_logs_event_id ON attendance_logs(event_id);

-- ============================================================
-- JOBS
-- ============================================================

CREATE TABLE jobs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    status      varchar NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'in_process', 'done', 'failed')),
    retry_count integer NOT NULL DEFAULT 0,
    data        jsonb,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_jobs_status ON jobs(status);