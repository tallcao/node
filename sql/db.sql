CREATE TABLE converter(
    id integer primary key,
    sn text unique not null,
    -- 1: can-io, 2: can-485, 3: can-relay
    -- 4: lora-io, 5: lora-485, 6: lora-relay
    converter_type integer not null,
    can_no integer not null,
    guid text unique,
    device_type integer 

);

CREATE TABLE panel(
    id integer primary key,
    sn text unique not null
);


-- CREATE TABLE device(
--     id integer primary key,

--     converter_sn text unique not null,
--     -- 0: can, 1: lora
--     converter_type integer not null
-- );


CREATE TABLE serial(
    id integer primary key,
    addr integer unique not null,  
    device_type integer,
    guid text
    

);

