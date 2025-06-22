
migrate:
	migrate -database "sqlite3://db.sqlite" -path ./migrations up

downgrade:
	migrate -database "sqlite3://db.sqlite" -path ./migrations down