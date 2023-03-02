CREATE TABLE files (
	hash_sum     TEXT UNIQUE, /* Контрольная сумма файла */
	orig_name    TEXT,        /* Оригинальное имя файла */
	file_id      TEXT UNIQUE, /* Идентификатор файла, ссылка для скачивания */
	size         INTEGER      /* Размер файла */
);
