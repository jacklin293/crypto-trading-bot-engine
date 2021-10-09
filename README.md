# crypto-trading-bot-engine


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

# Contract Strategy Params

* time should follow the format `RFC3339`

```
{
  "entry_type": "limit",
  "entry_order": {
    "trigger": {
      "trigger_type": "limit",
      "operator": "<=",
      "price": "48800"
    },
    "flip_operator_enabled": true
  },
  "stop_loss_order": {
    "trigger": {
      "trigger_type": "limit",
      "operator": "<=",
      "price": "40000"
    }
  },
  "take_profit_order": {
    "trigger": {
      "trigger_type": "limit",
      "operator": ">=",
      "price": "50000"
    }
  }
}
```

```
{
  "entry_type": "trendline",
  "entry_order": {
    "trendline_trigger": {
      "trigger_type": "line",
      "operator": ">=",
      "time_1": "2021-09-07T00:00:00Z",
      "price_1": "52920",
      "time_2": "2021-09-15T04:00:00Z",
      "price_2": "47221.54"
    },
    "trendline_offset_percent": 0.005
  },
  "stop_loss_order": {
    "loss_tolerance_percent": 0.005,
    "trendline_readjustment_enabled": true
  },
  "take_profit_order": {
    "trigger": {
      "trigger_type": "limit",
      "operator": ">=",
      "price": "50000"
    }
  }
}
```
