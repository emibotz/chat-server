
CREATE TABLE users (
    id uuid primary key not null default uuidv7(),
    name text unique not null,
    auth text not null
);
