run:
	go build && ./crypto-trading-bot-main
test:
	go test -count=1 ./...
dump-db-schemas:
	mysqldump -h 127.0.0.1 -u root -proot --no-data crypto users | sed -e 's/AUTO_INCREMENT=[[:digit:]]* //' > db_schemas/users.sql
	mysqldump -h 127.0.0.1 -u root -proot --no-data crypto contract_strategies | sed -e 's/AUTO_INCREMENT=[[:digit:]]* //' > db_schemas/contract_strategies.sql
