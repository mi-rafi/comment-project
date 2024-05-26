CREATE TABLE IF NOT EXISTS Post (
    id_p serial PRIMARY KEY,
    author  varchar(20) NOT NULL CONSTRAINT non_empty_name CHECK(length(author)>0),
    title varchar(100) NOT NULL CONSTRAINT non_empty_name CHECK(length(title)>0),
    text_p text NOT NULL CONSTRAINT non_empty_text CHECK(length(text_p)>0),
    comm boolean NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS Comment (
    id_c serial PRIMARY KEY,
    id_p int REFERENCES Post(id_p),
    parent int,
    author  varchar(20) NOT NULL CONSTRAINT non_empty_name CHECK(length(author)>0),
    text_c varchar(2000) NOT NULL CONSTRAINT non_empty_text CHECK(length(text_c)>0),
);