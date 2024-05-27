CREATE TABLE IF NOT EXISTS Post (
    id_p bigserial PRIMARY KEY,
    author  varchar(20) NOT NULL CONSTRAINT non_empty_author CHECK(length(author)>0),
    title varchar(100) NOT NULL CONSTRAINT non_empty_title CHECK(length(title)>0),
    text_p text NOT NULL CONSTRAINT non_empty_text CHECK(length(text_p)>0),
    comm boolean NOT NULL DEFAULT true,
    time_p TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS Comment (
    id_c bigserial PRIMARY KEY,
    id_p bigint REFERENCES Post(id_p),
    parent bigint,
    author  varchar(20) NOT NULL CONSTRAINT non_empty_name CHECK(length(author)>0),
    text_c varchar(2000) NOT NULL CONSTRAINT non_empty_text CHECK(length(text_c)>0), 
    time_c TIMESTAMP
);