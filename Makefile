run:
	go build && ./crypto-trading-bot-engine
test:
	go test -count=1 ./...
dump:
	# column-statistics is a new flag that is enabled by default in mysqldump 8.
	mysqldump -h 127.0.0.1 -u root -proot --column-statistics=0 --no-data crypto symbols | sed -e 's/AUTO_INCREMENT=[[:digit:]]* //' > db_schemas/symbols.sql
	mysqldump -h 127.0.0.1 -u root -proot --column-statistics=0 --no-data crypto users | sed -e 's/AUTO_INCREMENT=[[:digit:]]* //' > db_schemas/users.sql
	mysqldump -h 127.0.0.1 -u root -proot --column-statistics=0 --no-data crypto contract_strategies | sed -e 's/AUTO_INCREMENT=[[:digit:]]* //' > db_schemas/contract_strategies.sql
