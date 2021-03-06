DROP TABLE CONCRETE_TYPES CASCADE;
DROP TABLE INTERFACES CASCADE;
DROP TABLE CONCRETE_METHODS;
DROP TABLE INTERFACE_METHODS;

CREATE TABLE IF NOT EXISTS CONCRETE_TYPES (
    ID serial PRIMARY KEY,
    PACKAGE VARCHAR (255),
    NAME VARCHAR (255),
    BASETYPE VARCHAR (255)
);

CREATE TABLE IF NOT EXISTS CONCRETE_METHODS (
    ID serial PRIMARY KEY,
    PACKAGE VARCHAR (255),
    NAME VARCHAR (255),
    PARAMETERS VARCHAR (255)[],
    RETURN_VALUES VARCHAR (255)[],
    RECEIVER_NAME VARCHAR (255),
    RECEIVER_ID INTEGER REFERENCES CONCRETE_TYPES(ID) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS INTERFACES (
    ID serial PRIMARY KEY,
    PACKAGE VARCHAR (255),
    NAME VARCHAR (255),
    IMPLEMENTABLE BOOLEAN
);

CREATE TABLE IF NOT EXISTS INTERFACE_METHODS (
    ID serial PRIMARY KEY,
    PACKAGE VARCHAR (255),
    NAME VARCHAR (255),
    PARAMETERS VARCHAR (255)[],
    RETURN_VALUES VARCHAR (255)[],
    RECEIVER_NAME VARCHAR(255),
    RECEIVER_ID INTEGER REFERENCES INTERFACES(ID) ON DELETE CASCADE
)