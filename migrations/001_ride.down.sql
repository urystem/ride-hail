begin;

-- Drop ride events first (depends on ride_event_type and rides)
drop table if exists ride_events;

-- Drop ride_event_type enumeration
drop table if exists ride_event_type;

-- Drop rides (depends on coordinates, users, ride_status, vehicle_type)
drop table if exists rides;

-- Drop coordinates (standalone)
drop table if exists coordinates;

-- Drop vehicle_type enumeration
drop table if exists vehicle_type;

-- Drop ride_status enumeration
drop table if exists ride_status;

-- Drop users (depends on roles, user_status)
drop table if exists users;

-- Drop user status enumeration
drop table if exists user_status;

-- Drop roles enumeration
drop table if exists roles;

commit;
