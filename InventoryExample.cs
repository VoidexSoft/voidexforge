using System;
using System.Collections.Generic;
using System.Threading.Tasks;
using Nakama;
using VoidexForge.Client.Models;
using VoidexForge.Client.Services;

namespace VoidexForge.Client.Examples
{
    /// <summary>
    /// Example usage of the inventory system
    /// </summary>
    public class InventoryExample
    {
        private readonly InventoryService _inventoryService;

        public InventoryExample(IClient client, ISession session)
        {
            _inventoryService = new InventoryService(client, session);
        }

        /// <summary>
        /// Example: Load and display player's inventory
        /// </summary>
        public async Task LoadPlayerInventoryExample()
        {
            try
            {
                Console.WriteLine("Loading player inventory...");
                
                // Get all inventory items
                var inventory = await _inventoryService.ListPlayerInventoryAsync();
                
                Console.WriteLine($"Player has {inventory.Items.Count} different items:");
                
                foreach (var kvp in inventory.Items)
                {
                    var item = kvp.Value;
                    Console.WriteLine($"- {item.Name} (ID: {item.Id}): {item.Count}/{item.MaxCount} " +
                                    $"[Category: {item.Category}, Consumable: {item.Consumable}]");
                    
                    // Show when the item was acquired
                    Console.WriteLine($"  Acquired: {item.OwnedTime:yyyy-MM-dd HH:mm:ss}");
                    
                    // Show custom properties if any
                    if (item.StringProperties.Count > 0)
                    {
                        Console.WriteLine("  String Properties:");
                        foreach (var prop in item.StringProperties)
                        {
                            Console.WriteLine($"    {prop.Key}: {prop.Value}");
                        }
                    }
                    
                    if (item.NumericProperties.Count > 0)
                    {
                        Console.WriteLine("  Numeric Properties:");
                        foreach (var prop in item.NumericProperties)
                        {
                            Console.WriteLine($"    {prop.Key}: {prop.Value}");
                        }
                    }
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error loading inventory: {ex.Message}");
            }
        }

        /// <summary>
        /// Example: Filter inventory by category
        /// </summary>
        public async Task FilterByCategoryExample()
        {
            try
            {
                Console.WriteLine("Loading weapons from inventory...");
                
                // Get only weapons
                var weaponsInventory = await _inventoryService.ListPlayerInventoryAsync("weapons");
                var weapons = InventoryService.GetItemsByCategory(weaponsInventory, "weapons");
                
                Console.WriteLine($"Player has {weapons.Count} weapons:");
                
                foreach (var weapon in weapons.Values)
                {
                    Console.WriteLine($"- {weapon.Name}: {weapon.Count} owned");
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error loading weapons: {ex.Message}");
            }
        }

        /// <summary>
        /// Example: Consume a health potion
        /// </summary>
        public async Task ConsumeHealthPotionExample()
        {
            try
            {
                Console.WriteLine("Attempting to consume health potion...");
                
                // Check if player has health potions
                var inventory = await _inventoryService.ListPlayerInventoryAsync();
                
                if (InventoryService.HasEnoughItems(inventory, "health_potion", 1))
                {
                    // Consume one health potion
                    var result = await _inventoryService.ConsumeItemAsync("health_potion", 1);
                    
                    Console.WriteLine("Health potion consumed successfully!");
                    
                    // Check if we got any rewards
                    if (result.Rewards.Count > 0)
                    {
                        Console.WriteLine("Rewards received:");
                        foreach (var reward in result.Rewards)
                        {
                            Console.WriteLine($"- Reward for {reward.Key}");
                        }
                    }
                    
                    // Show updated inventory count
                    if (result.Inventory?.Items.TryGetValue("health_potion", out var updatedItem) == true)
                    {
                        Console.WriteLine($"Health potions remaining: {updatedItem.Count}");
                    }
                }
                else
                {
                    Console.WriteLine("No health potions available to consume!");
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error consuming health potion: {ex.Message}");
            }
        }

        /// <summary>
        /// Example: Update item properties (e.g., enchantment level)
        /// </summary>
        public async Task UpdateItemPropertiesExample()
        {
            try
            {
                Console.WriteLine("Updating sword enchantment...");
                
                // Get player's inventory to find a sword instance
                var inventory = await _inventoryService.ListPlayerInventoryAsync();
                
                InventoryItem? sword = null;
                foreach (var item in inventory.Items.Values)
                {
                    if (item.Category == "weapons" && item.Name.Contains("Sword") && !string.IsNullOrEmpty(item.InstanceId))
                    {
                        sword = item;
                        break;
                    }
                }
                
                if (sword != null)
                {
                    // Update the sword's enchantment level
                    var stringProps = new Dictionary<string, string>
                    {
                        { "enchantment", "fire" },
                        { "rarity", "legendary" }
                    };
                    
                    var numericProps = new Dictionary<string, double>
                    {
                        { "enchantment_level", 5.0 },
                        { "damage_bonus", 25.5 }
                    };
                    
                    var result = await _inventoryService.UpdateItemPropertiesAsync(
                        sword.InstanceId, 
                        stringProps, 
                        numericProps);
                    
                    Console.WriteLine("Sword enchantment updated successfully!");
                    
                    // Show updated item if returned
                    if (result.Inventory?.Items.TryGetValue(sword.Id, out var updatedSword) == true)
                    {
                        Console.WriteLine($"Updated sword properties:");
                        foreach (var prop in updatedSword.StringProperties)
                        {
                            Console.WriteLine($"  {prop.Key}: {prop.Value}");
                        }
                        foreach (var prop in updatedSword.NumericProperties)
                        {
                            Console.WriteLine($"  {prop.Key}: {prop.Value}");
                        }
                    }
                }
                else
                {
                    Console.WriteLine("No enchantable sword found in inventory!");
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error updating item properties: {ex.Message}");
            }
        }

        /// <summary>
        /// Example: Check if player can afford a recipe
        /// </summary>
        public async Task CheckRecipeAffordabilityExample()
        {
            try
            {
                Console.WriteLine("Checking if player can craft Super Health Potion...");
                
                var inventory = await _inventoryService.ListPlayerInventoryAsync();
                
                // Recipe requirements
                var recipe = new Dictionary<string, long>
                {
                    { "health_potion", 3 },
                    { "magic_essence", 1 },
                    { "crystal_shard", 2 }
                };
                
                bool canCraft = true;
                var missingItems = new List<string>();
                
                foreach (var ingredient in recipe)
                {
                    var required = ingredient.Value;
                    var available = InventoryService.GetItemTotalCount(inventory, ingredient.Key);
                    
                    if (available < required)
                    {
                        canCraft = false;
                        missingItems.Add($"{ingredient.Key} (need {required}, have {available})");
                    }
                    else
                    {
                        Console.WriteLine($"✓ {ingredient.Key}: {available}/{required}");
                    }
                }
                
                if (canCraft)
                {
                    Console.WriteLine("✅ Player can craft Super Health Potion!");
                }
                else
                {
                    Console.WriteLine("❌ Cannot craft Super Health Potion. Missing:");
                    foreach (var missing in missingItems)
                    {
                        Console.WriteLine($"  - {missing}");
                    }
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error checking recipe: {ex.Message}");
            }
        }

        /// <summary>
        /// Example: Display all consumable items
        /// </summary>
        public async Task ShowConsumableItemsExample()
        {
            try
            {
                Console.WriteLine("Loading consumable items...");
                
                var inventory = await _inventoryService.ListPlayerInventoryAsync();
                var consumables = InventoryService.GetConsumableItems(inventory);
                
                if (consumables.Count > 0)
                {
                    Console.WriteLine($"Player has {consumables.Count} types of consumable items:");
                    
                    foreach (var consumable in consumables.Values)
                    {
                        Console.WriteLine($"- {consumable.Name}: {consumable.Count} available");
                        
                        if (consumable.ConsumeAvailableRewards != null)
                        {
                            Console.WriteLine($"  Grants rewards when consumed");
                        }
                    }
                }
                else
                {
                    Console.WriteLine("No consumable items found in inventory.");
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error loading consumables: {ex.Message}");
            }
        }

        /// <summary>
        /// Example: Bulk operations
        /// </summary>
        public async Task BulkOperationsExample()
        {
            try
            {
                Console.WriteLine("Performing bulk inventory operations...");
                
                // Grant multiple items at once
                var itemsToGrant = new Dictionary<string, long>
                {
                    { "health_potion", 5 },
                    { "mana_potion", 3 },
                    { "gold_coin", 100 }
                };
                
                var grantResult = await _inventoryService.GrantItemsAsync(itemsToGrant);
                Console.WriteLine("Items granted successfully!");
                
                // Consume multiple items at once
                var itemsToConsume = new Dictionary<string, long>
                {
                    { "health_potion", 2 },
                    { "mana_potion", 1 }
                };
                
                var consumeResult = await _inventoryService.ConsumeItemsAsync(itemsToConsume);
                Console.WriteLine("Items consumed successfully!");
                
                // Show any rewards from consumption
                if (consumeResult.Rewards.Count > 0)
                {
                    Console.WriteLine("Consumption rewards:");
                    foreach (var reward in consumeResult.Rewards)
                    {
                        Console.WriteLine($"- Rewards from {reward.Key}");
                    }
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error in bulk operations: {ex.Message}");
            }
        }
    }
} 