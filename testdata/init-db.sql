CREATE TABLE IF NOT EXISTS Post (
    id_p bigserial PRIMARY KEY,
    author  varchar(20) NOT NULL CONSTRAINT non_empty_author CHECK(length(author)>0),
    title varchar(100) NOT NULL CONSTRAINT non_empty_title CHECK(length(title)>0),
    text_p text NOT NULL CONSTRAINT non_empty_text CHECK(length(text_p)>0),
    comm boolean NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS Comment (
    id_c bigserial PRIMARY KEY,
    id_p bigint REFERENCES Post(id_p),
    parent bigint,
    author  varchar(20) NOT NULL CONSTRAINT non_empty_name CHECK(length(author)>0),
    text_c varchar(2000) NOT NULL CONSTRAINT non_empty_text CHECK(length(text_c)>0)
);

-- INSERT INTO Post (author, title, text_p, comm) VALUES ($1, $2, $3, $4)

-- INSERT INTO Comment (id_p, parent, author, text_c) VALUES ($1, $2, $3, $4)


-- SELECT author, title, text_p
-- FROM Post
-- WHERE id_p = $1 

-- SELECT id_p, author, title, text_p, comm 
-- FROM Post 
-- ORDER BY title
-- LIMIT $1
-- OFFSET $2

-- SELECT  comm 
-- FROM Post 
-- WHERE id_p = $1

-- SELECT  id_p, parent, author, text_c
-- FROM Comment 
-- WHERE id_p = $1



-- WITH RECURSIVE comment_tree AS (
--     SELECT
--         id_c,
--         parent,
--         author, 
--         text_c,
--         1 AS level,
--         ARRAY[id] AS path
--     FROM comments
--     WHERE parent_id IS NULL

--     UNION ALL

--     SELECT
--         c.id_c,
--         c.parent,
--         author, 
--         text_c,
--         ct.level + 1 AS level,
--         ct.path || c.id
--     FROM comments c
--     INNER JOIN comment_tree ct ON c.parent = ct.id_c
-- ),
-- numbered_comments AS (
--     SELECT *,
--            ROW_NUMBER() OVER (ORDER BY path) AS row_num
--     FROM comment_tree
-- )
-- SELECT *
-- FROM numbered_comments
-- WHERE row_num > $1
-- ORDER BY row_num
-- LIMIT $2;