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

INSERT INTO Post (author, title, text_p, comm) VALUES ($1, $2, $3, $4)

INSERT INTO Comment (id_p, parent, author, text_c) VALUES ($1, $2, $3, $4)


SELECT author, title, text_p
FROM Post
WHERE id_p = $1 

SELECT id_p, author, title, text_p, comm 
FROM Post 
ORDER BY title
LIMIT $1
OFFSET $2

SELECT  author, title, text_p, comm 
FROM Post 
WHERE id_p = $1