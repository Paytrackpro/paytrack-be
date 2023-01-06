CREATE TABLE public.users (
    id uuid NOT NULL DEFAULT uuid_generate_v4() not null,
    user_name varchar NOT NULL,
    email varchar NOT NULL,
    password_hash varchar NOT NULL,
    created_at timestamp NULL,
    updated_at timestamp NULL
);