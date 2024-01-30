CREATE TABLE IF NOT EXISTS "files" (
	"name"      TEXT UNIQUE, /* Имя файла */
	"orig_name" TEXT,        /* Оригинальное имя файла */
	"size"      INTEGER      /* Размер файла */
);
