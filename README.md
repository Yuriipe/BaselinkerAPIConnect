BaselinkerAPIConnect uses simple order quantity based logic to bulk modify product prices.

For usage Mongo DB should be set up.

App gathers products information from Baselinker CRM and creates coresponding documents with product id's, prices, stock quantity and orders from last 7 days (period can be changed in [config/payloadCfg.env], [GO_DAYSBEFORE] - parameter)

You can easily change price-update business logic based on sales results from the past.
CLI menu allows to choose required action.

Next version:
- cron-friendly half automatic execution
- logs
- product order long and short-time reports 