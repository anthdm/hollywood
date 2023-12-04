# Trade-Engine Example

**Note:** This is a conceptual example. It does not execute actual trades.

## Overview

This example outlines a simple trade engine system composed of three distinct types of actors. Each actor plays a specific role in the system to manage and execute trades efficiently.

### Actor Types

1. **Trade Engine Actor**

   - **Role:** This singular actor is responsible for creating and overseeing the Price Watcher and Trade Executor actors.
   - **Functionality:** It acts as a central management hub for the system.

2. **Price Watcher**

   - **Role:** Dedicated to monitoring the price of a specific ticker. The system creates one Price Watcher per ticker to optimize resource usage.
   - **Key Functions:**
     - **Periodic Updates:** Utilizes `(*actor.Engine).SendRepeat` to regularly send an `TriggerPriceUpdate` message to itself.
     - **Price Update:** Upon receiving an `TriggerPriceUpdate` message, it updates the latest price and sends all subscribed Trade Executors a `PriceUpdate` message.
     - **Subscriber Check:** During each `TriggerPriceUpdate`, it checks for active subscribers. If no subscribers exist, it uses `(actor.SendRepeater).Stop()` to stop the `SendRepeat` and then posions itself using `(*actor.Engine).Poison`.

3. **Trade Executor**
   - **Role:** Manages the execution of individual trades.
   - **Key Functions:**
     - **Subscription:** Subscribes to the Price Watcher with the ticker it is trading.
     - **Trade Decision:** On receiving a `PriceUpdate`, it checks if the current price falls within its trade parameters.(not implemented in example - just prints the price)
     - **Cancellation:** If a trade is canceled, it sends an `Unsubscribe` message to the Price Watcher and then poisons itself using `(*actor.Engine).Poison`.

### Flow

1. The **Trade Engine Actor** sets up the system by creating Price Watcher and Trade Executor actors.
2. **Price Watchers** continuously monitor prices, updating subscribed Trade Executors.
3. **Trade Executors** analyze these updates and execute trades based on the updates.
4. If a trade is canceled or completed, the corresponding Trade Executor unsubscribes from its Price Watcher and poisons itself.
