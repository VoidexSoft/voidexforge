using System;
using System.Collections.Generic;
using System.Threading.Tasks;
using System.Text.Json;
using Nakama;
using VoidexForge.Client.Models;

namespace VoidexForge.Client.Services
{
    /// <summary>
    /// Service for managing inventory operations with the server
    /// </summary>
    public class InventoryService
    {
        private readonly IClient _client;
        private readonly ISession _session;

        public InventoryService(IClient client, ISession session)
        {
            _client = client ?? throw new ArgumentNullException(nameof(client));
            _session = session ?? throw new ArgumentNullException(nameof(session));
        }

        /// <summary>
        /// List all inventory items defined in the codex
        /// </summary>
        /// <param name="category">Optional category filter</param>
        /// <returns>List of available inventory items</returns>
        public async Task<InventoryList> ListInventoryDefinitionsAsync(string category = "")
        {
            var request = new InventoryListRequest
            {
                ItemCategory = category
            };

            var requestJson = JsonSerializer.Serialize(request);
            var rpcResponse = await _client.RpcAsync(_session, "rpc_inventory_list", requestJson);
            
            return JsonSerializer.Deserialize<InventoryList>(rpcResponse.Payload)
                ?? throw new InvalidOperationException("Failed to deserialize inventory list response");
        }

        /// <summary>
        /// List all inventory items owned by the player
        /// </summary>
        /// <param name="category">Optional category filter</param>
        /// <returns>Player's inventory items</returns>
        public async Task<InventoryList> ListPlayerInventoryAsync(string category = "")
        {
            var request = new InventoryListRequest
            {
                ItemCategory = category
            };

            var requestJson = JsonSerializer.Serialize(request);
            var rpcResponse = await _client.RpcAsync(_session, "rpc_inventory_list_inventory", requestJson);
            
            return JsonSerializer.Deserialize<InventoryList>(rpcResponse.Payload)
                ?? throw new InvalidOperationException("Failed to deserialize player inventory response");
        }

        /// <summary>
        /// Grant items to the player
        /// </summary>
        /// <param name="items">Dictionary of item IDs and quantities to grant</param>
        /// <returns>Updated inventory acknowledgment</returns>
        public async Task<InventoryUpdateAck> GrantItemsAsync(Dictionary<string, long> items)
        {
            var request = new InventoryGrantRequest
            {
                Items = items
            };

            var requestJson = JsonSerializer.Serialize(request);
            var rpcResponse = await _client.RpcAsync(_session, "rpc_inventory_grant", requestJson);
            
            return JsonSerializer.Deserialize<InventoryUpdateAck>(rpcResponse.Payload)
                ?? throw new InvalidOperationException("Failed to deserialize inventory grant response");
        }

        /// <summary>
        /// Grant a single item to the player
        /// </summary>
        /// <param name="itemId">The item ID to grant</param>
        /// <param name="quantity">The quantity to grant</param>
        /// <returns>Updated inventory acknowledgment</returns>
        public async Task<InventoryUpdateAck> GrantItemAsync(string itemId, long quantity = 1)
        {
            var items = new Dictionary<string, long> { { itemId, quantity } };
            return await GrantItemsAsync(items);
        }

        /// <summary>
        /// Consume items from the player's inventory
        /// </summary>
        /// <param name="items">Dictionary of item IDs and quantities to consume</param>
        /// <param name="allowOverconsume">Whether to allow consuming more than owned</param>
        /// <param name="instances">Dictionary of instance IDs and quantities to consume</param>
        /// <returns>Consumption result with updated inventory and rewards</returns>
        public async Task<InventoryConsumeRewards> ConsumeItemsAsync(
            Dictionary<string, long>? items = null,
            bool allowOverconsume = false,
            Dictionary<string, long>? instances = null)
        {
            var request = new InventoryConsumeRequest
            {
                Items = items ?? new Dictionary<string, long>(),
                Overconsume = allowOverconsume,
                Instances = instances ?? new Dictionary<string, long>()
            };

            var requestJson = JsonSerializer.Serialize(request);
            var rpcResponse = await _client.RpcAsync(_session, "rpc_inventory_consume", requestJson);
            
            return JsonSerializer.Deserialize<InventoryConsumeRewards>(rpcResponse.Payload)
                ?? throw new InvalidOperationException("Failed to deserialize inventory consume response");
        }

        /// <summary>
        /// Consume a single item from the player's inventory
        /// </summary>
        /// <param name="itemId">The item ID to consume</param>
        /// <param name="quantity">The quantity to consume</param>
        /// <param name="allowOverconsume">Whether to allow consuming more than owned</param>
        /// <returns>Consumption result with updated inventory and rewards</returns>
        public async Task<InventoryConsumeRewards> ConsumeItemAsync(
            string itemId, 
            long quantity = 1, 
            bool allowOverconsume = false)
        {
            var items = new Dictionary<string, long> { { itemId, quantity } };
            return await ConsumeItemsAsync(items, allowOverconsume);
        }

        /// <summary>
        /// Update properties of instanced inventory items
        /// </summary>
        /// <param name="itemUpdates">Dictionary of instance IDs and their property updates</param>
        /// <returns>Updated inventory acknowledgment</returns>
        public async Task<InventoryUpdateAck> UpdateItemPropertiesAsync(
            Dictionary<string, InventoryUpdateItemProperties> itemUpdates)
        {
            var request = new InventoryUpdateItemsRequest
            {
                ItemUpdates = itemUpdates
            };

            var requestJson = JsonSerializer.Serialize(request);
            var rpcResponse = await _client.RpcAsync(_session, "rpc_inventory_update", requestJson);
            
            return JsonSerializer.Deserialize<InventoryUpdateAck>(rpcResponse.Payload)
                ?? throw new InvalidOperationException("Failed to deserialize inventory update response");
        }

        /// <summary>
        /// Update properties of a single instanced inventory item
        /// </summary>
        /// <param name="instanceId">The instance ID of the item to update</param>
        /// <param name="stringProperties">String properties to update</param>
        /// <param name="numericProperties">Numeric properties to update</param>
        /// <returns>Updated inventory acknowledgment</returns>
        public async Task<InventoryUpdateAck> UpdateItemPropertiesAsync(
            string instanceId,
            Dictionary<string, string>? stringProperties = null,
            Dictionary<string, double>? numericProperties = null)
        {
            var properties = new InventoryUpdateItemProperties
            {
                StringProperties = stringProperties ?? new Dictionary<string, string>(),
                NumericProperties = numericProperties ?? new Dictionary<string, double>()
            };

            var itemUpdates = new Dictionary<string, InventoryUpdateItemProperties>
            {
                { instanceId, properties }
            };

            return await UpdateItemPropertiesAsync(itemUpdates);
        }

        /// <summary>
        /// Helper method to find items by category in an inventory
        /// </summary>
        /// <param name="inventory">The inventory to search</param>
        /// <param name="category">The category to filter by</param>
        /// <returns>Dictionary of items in the specified category</returns>
        public static Dictionary<string, InventoryItem> GetItemsByCategory(
            InventoryList inventory, 
            string category)
        {
            var result = new Dictionary<string, InventoryItem>();
            
            foreach (var kvp in inventory.Items)
            {
                if (string.Equals(kvp.Value.Category, category, StringComparison.OrdinalIgnoreCase))
                {
                    result[kvp.Key] = kvp.Value;
                }
            }
            
            return result;
        }

        /// <summary>
        /// Helper method to get total count of a specific item across all instances
        /// </summary>
        /// <param name="inventory">The inventory to search</param>
        /// <param name="itemId">The item ID to count</param>
        /// <returns>Total count of the item</returns>
        public static long GetItemTotalCount(InventoryList inventory, string itemId)
        {
            if (inventory.Items.TryGetValue(itemId, out var item))
            {
                return item.Count;
            }
            return 0;
        }

        /// <summary>
        /// Helper method to check if player has enough of an item
        /// </summary>
        /// <param name="inventory">The inventory to check</param>
        /// <param name="itemId">The item ID to check</param>
        /// <param name="requiredQuantity">The required quantity</param>
        /// <returns>True if player has enough of the item</returns>
        public static bool HasEnoughItems(InventoryList inventory, string itemId, long requiredQuantity)
        {
            return GetItemTotalCount(inventory, itemId) >= requiredQuantity;
        }

        /// <summary>
        /// Helper method to get all consumable items from inventory
        /// </summary>
        /// <param name="inventory">The inventory to search</param>
        /// <returns>Dictionary of consumable items</returns>
        public static Dictionary<string, InventoryItem> GetConsumableItems(InventoryList inventory)
        {
            var result = new Dictionary<string, InventoryItem>();
            
            foreach (var kvp in inventory.Items)
            {
                if (kvp.Value.Consumable && kvp.Value.Count > 0)
                {
                    result[kvp.Key] = kvp.Value;
                }
            }
            
            return result;
        }
    }
} 