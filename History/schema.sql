CREATE user 'greg'@'%' IDENTIFIED BY 'root';
GRANT ALL PRIVILEGES ON * . * TO 'greg'@'%';

CREATE DATABASE nerds;
USE nerds;

CREATE TABLE authors (
  id   BIGINT  NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name text    NOT NULL,
  bio  text
);

INSERT INTO authors (name, bio) VALUES 
('Virginia Woolf', 'Modernist English writer and essayist, known for Mrs Dalloway and A Room of One''s Own'),
('Gabriel García Márquez', 'Colombian novelist, known for One Hundred Years of Solitude and magical realism'),
('Haruki Murakami', 'Contemporary Japanese author blending surrealism with pop culture'),
('Chimamanda Ngozi Adichie', 'Nigerian writer exploring themes of identity and feminism in contemporary Africa'),
('Jorge Luis Borges', 'Argentine short-story writer known for complex philosophical narratives'),
('Toni Morrison', 'Nobel Prize-winning American novelist exploring African-American experience'),
('Italo Calvino', 'Italian writer known for postmodern fiction and fantasy works'),
('Margaret Atwood', 'Canadian author known for speculative fiction and poetry'),
('Umberto Eco', 'Italian novelist and philosopher, author of complex historical mysteries'),
('Zadie Smith', 'British novelist examining multicultural London life and identity');