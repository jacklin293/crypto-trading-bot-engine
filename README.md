# crypto-trading-bot-main


# Set up the environment

### 1. Set up database

```
docker-compose up -d
```

> Folder `mariadb` will be created, so that you won't miss any data when db container gets killed.

### 2. Check if database is up

> You might need to change the port in `docker-compose.yml` if the port conflicts with the port that you're using locally

Connect to database directly

```
mysql -h 127.0.0.1 -u root -proot
```

Or use phpmyadmin (if no error is shown on the page, it means fine)

```
http://localhost:8080/
```
