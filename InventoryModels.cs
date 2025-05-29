using System;
using System.Collections.Generic;
using System.Text.Json.Serialization;

namespace VoidexForge.Client.Models
{
    /// <summary>
    /// Represents an inventory item in the game system
    /// </summary>
    public class InventoryItem
    {
        /// <summary>
        /// The unique identifier of the item
        /// </summary>
        [JsonPropertyName("id")]
        public string Id { get; set; } = string.Empty;

        /// <summary>
        /// The display name of the item (may be an i18n code)
        /// </summary>
        [JsonPropertyName("name")]
        public string Name { get; set; } = string.Empty;

        /// <summary>
        /// A description of the item (may be an i18n code)
        /// </summary>
        [JsonPropertyName("description")]
        public string Description { get; set; } = string.Empty;

        /// <summary>
        /// The category to group the item with others
        /// </summary>
        [JsonPropertyName("category")]
        public string Category { get; set; } = string.Empty;

        /// <summary>
        /// The sets the item is grouped into
        /// </summary>
        [JsonPropertyName("item_sets")]
        public List<string> ItemSets { get; set; } = new List<string>();

        /// <summary>
        /// The current count/quantity of the item
        /// </summary>
        [JsonPropertyName("count")]
        public long Count { get; set; }

        /// <summary>
        /// The maximum count which can be owned for this item
        /// </summary>
        [JsonPropertyName("max_count")]
        public long MaxCount { get; set; }

        /// <summary>
        /// Whether or not the item is stackable
        /// </summary>
        [JsonPropertyName("stackable")]
        public bool Stackable { get; set; }

        /// <summary>
        /// Whether or not the item is consumable
        /// </summary>
        [JsonPropertyName("consumable")]
        public bool Consumable { get; set; }

        /// <summary>
        /// The configuration for rewards granted upon consumption
        /// </summary>
        [JsonPropertyName("consume_available_rewards")]
        public AvailableRewards? ConsumeAvailableRewards { get; set; }

        /// <summary>
        /// The properties with string values
        /// </summary>
        [JsonPropertyName("string_properties")]
        public Dictionary<string, string> StringProperties { get; set; } = new Dictionary<string, string>();

        /// <summary>
        /// The properties with numeric values
        /// </summary>
        [JsonPropertyName("numeric_properties")]
        public Dictionary<string, double> NumericProperties { get; set; } = new Dictionary<string, double>();

        /// <summary>
        /// UNIX timestamp when the user acquired this item
        /// </summary>
        [JsonPropertyName("owned_time_sec")]
        public long OwnedTimeSec { get; set; }

        /// <summary>
        /// UNIX timestamp when the item was last updated
        /// </summary>
        [JsonPropertyName("update_time_sec")]
        public long UpdateTimeSec { get; set; }

        /// <summary>
        /// The instance ID of the item, if any
        /// </summary>
        [JsonPropertyName("instance_id")]
        public string InstanceId { get; set; } = string.Empty;

        /// <summary>
        /// Convenience property to get the owned time as DateTime
        /// </summary>
        [JsonIgnore]
        public DateTime OwnedTime => DateTimeOffset.FromUnixTimeSeconds(OwnedTimeSec).DateTime;

        /// <summary>
        /// Convenience property to get the update time as DateTime
        /// </summary>
        [JsonIgnore]
        public DateTime UpdateTime => DateTimeOffset.FromUnixTimeSeconds(UpdateTimeSec).DateTime;
    }

    /// <summary>
    /// Represents the player's complete inventory
    /// </summary>
    public class Inventory
    {
        /// <summary>
        /// The items in the player's inventory, keyed by item ID
        /// </summary>
        [JsonPropertyName("items")]
        public Dictionary<string, InventoryItem> Items { get; set; } = new Dictionary<string, InventoryItem>();
    }

    /// <summary>
    /// Request to list inventory items
    /// </summary>
    public class InventoryListRequest
    {
        /// <summary>
        /// The category for items to filter for, or empty for all
        /// </summary>
        [JsonPropertyName("item_category")]
        public string ItemCategory { get; set; } = string.Empty;
    }

    /// <summary>
    /// Response containing inventory items list
    /// </summary>
    public class InventoryList
    {
        /// <summary>
        /// The inventory items from definitions and the user
        /// </summary>
        [JsonPropertyName("items")]
        public Dictionary<string, InventoryItem> Items { get; set; } = new Dictionary<string, InventoryItem>();
    }

    /// <summary>
    /// Request to grant items to the user
    /// </summary>
    public class InventoryGrantRequest
    {
        /// <summary>
        /// The items to grant, keyed by item ID with quantities as values
        /// </summary>
        [JsonPropertyName("items")]
        public Dictionary<string, long> Items { get; set; } = new Dictionary<string, long>();
    }

    /// <summary>
    /// Request to consume items from inventory
    /// </summary>
    public class InventoryConsumeRequest
    {
        /// <summary>
        /// Item ID amounts to consume, if any
        /// </summary>
        [JsonPropertyName("items")]
        public Dictionary<string, long> Items { get; set; } = new Dictionary<string, long>();

        /// <summary>
        /// Whether or not to allow overconsumption
        /// </summary>
        [JsonPropertyName("overconsume")]
        public bool Overconsume { get; set; }

        /// <summary>
        /// Instance ID amounts to consume, if any
        /// </summary>
        [JsonPropertyName("instances")]
        public Dictionary<string, long> Instances { get; set; } = new Dictionary<string, long>();
    }

    /// <summary>
    /// Properties to update on an instanced inventory item
    /// </summary>
    public class InventoryUpdateItemProperties
    {
        /// <summary>
        /// The properties with string values to update
        /// </summary>
        [JsonPropertyName("string_properties")]
        public Dictionary<string, string> StringProperties { get; set; } = new Dictionary<string, string>();

        /// <summary>
        /// The properties with numeric values to update
        /// </summary>
        [JsonPropertyName("numeric_properties")]
        public Dictionary<string, double> NumericProperties { get; set; } = new Dictionary<string, double>();
    }

    /// <summary>
    /// Request to update properties of instanced items
    /// </summary>
    public class InventoryUpdateItemsRequest
    {
        /// <summary>
        /// The item updates to action, keyed by item instance ID
        /// </summary>
        [JsonPropertyName("item_updates")]
        public Dictionary<string, InventoryUpdateItemProperties> ItemUpdates { get; set; } = new Dictionary<string, InventoryUpdateItemProperties>();
    }

    /// <summary>
    /// Response from inventory operations that modify inventory
    /// </summary>
    public class InventoryUpdateAck
    {
        /// <summary>
        /// Updated inventory data, if changed
        /// </summary>
        [JsonPropertyName("inventory")]
        public Inventory? Inventory { get; set; }
    }

    /// <summary>
    /// Response from consuming items, includes updated inventory and rewards
    /// </summary>
    public class InventoryConsumeRewards
    {
        /// <summary>
        /// Updated inventory data, if changed
        /// </summary>
        [JsonPropertyName("inventory")]
        public Inventory? Inventory { get; set; }

        /// <summary>
        /// Consume rewards by item ID, if any
        /// </summary>
        [JsonPropertyName("rewards")]
        public Dictionary<string, RewardList> Rewards { get; set; } = new Dictionary<string, RewardList>();

        /// <summary>
        /// Consume rewards by instance ID, if any
        /// </summary>
        [JsonPropertyName("instance_rewards")]
        public Dictionary<string, RewardList> InstanceRewards { get; set; } = new Dictionary<string, RewardList>();
    }

    /// <summary>
    /// Represents available rewards configuration
    /// </summary>
    public class AvailableRewards
    {
        // Add properties based on your reward system structure
        // This is a placeholder - you'll need to implement based on your reward system
    }

    /// <summary>
    /// Represents a list of rewards
    /// </summary>
    public class RewardList
    {
        // Add properties based on your reward system structure
        // This is a placeholder - you'll need to implement based on your reward system
    }

    /// <summary>
    /// RPC identifiers for inventory operations
    /// </summary>
    public enum InventoryRpcId
    {
        /// <summary>
        /// List all inventory items defined in the codex
        /// </summary>
        InventoryList = 1,

        /// <summary>
        /// List all inventory items owned by the player
        /// </summary>
        InventoryListInventory = 2,

        /// <summary>
        /// Consume one or more inventory items owned by the player
        /// </summary>
        InventoryConsume = 3,

        /// <summary>
        /// Grant one or more inventory items to the player
        /// </summary>
        InventoryGrant = 4,

        /// <summary>
        /// Update the properties on one or more inventory items owned by the player
        /// </summary>
        InventoryUpdate = 5
    }
} 