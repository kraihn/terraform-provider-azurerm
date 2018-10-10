package azurerm

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2017-10-01-preview/sql"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmSql2017ElasticPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmSql2017ElasticPoolCreate,
		Read:   resourceArmSql2017ElasticPoolRead,
		Update: resourceArmSql2017ElasticPoolCreate,
		Delete: resourceArmSql2017ElasticPoolDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

			"resource_group_name": resourceGroupNameSchema(),

			"server_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"sku": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
						},

						"capacity": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"size": {
							Type:             schema.TypeString,
							Optional:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
						},

						"tier": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
						},

						"family": {
							Type:             schema.TypeString,
							Optional:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
						},
					},
				},
			},

			"elastic_pool_properties": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"state": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"creation_date": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"max_size_bytes": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"per_database_settings": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"min_capacity": {
										Type:     schema.TypeFloat,
										Required: true,
									},

									"max_capacity": {
										Type:     schema.TypeFloat,
										Required: true,
									},
								},
							},
						},

						"zone_redundant": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"license_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmSql2017ElasticPoolCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).sql2017ElasticPoolsClient
	ctx := meta.(*ArmClient).StopContext

	log.Printf("[INFO] preparing arguments for SQL2017 ElasticPool creation.")

	elasticPoolName := d.Get("name").(string)
	serverName := d.Get("server_name").(string)
	location := azureRMNormalizeLocation(d.Get("location").(string))
	resGroup := d.Get("resource_group_name").(string)
	sku := expandAzureRmSql2017ElasticPoolSku(d)
	properties := expandAzureRmSql2017ElasticPoolProperties(d)
	tags := d.Get("tags").(map[string]interface{})

	elasticPool := sql.ElasticPool{
		Sku: sku,
		ElasticPoolProperties: properties,
		Location:              &location,
		Tags:                  expandTags(tags),
		Name:                  &elasticPoolName,
	}

	future, err := client.CreateOrUpdate(ctx, resGroup, serverName, elasticPoolName, elasticPool)
	if err != nil {
		return err
	}

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		return err
	}

	read, err := client.Get(ctx, resGroup, serverName, elasticPoolName)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read SQL2017 ElasticPool %q (resource group %q) ID", elasticPoolName, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmSql2017ElasticPoolRead(d, meta)
}

func resourceArmSql2017ElasticPoolRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).sql2017ElasticPoolsClient
	ctx := meta.(*ArmClient).StopContext

	resGroup, serverName, name, err := parseArmSql2017ElasticPoolId(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, resGroup, serverName, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Sql2017 Elastic Pool %s: %s", name, err)
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)

	if location := resp.Location; location != nil {
		d.Set("location", azureRMNormalizeLocation(*location))
	}

	d.Set("server_name", serverName)

	if err := d.Set("sku", flattenAzureRmSql2017ElasticPoolSku(resp.Sku)); err != nil {
		return fmt.Errorf("Error flattening `sku`: %+v", err)
	}

	if err := d.Set("elastic_pool_properties", flattenAzureRmSql2017ElasticPoolProperties(resp.ElasticPoolProperties)); err != nil {
		return fmt.Errorf("Error flattening `elastic_pool_properties`: %+v", err)
	}

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmSql2017ElasticPoolDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).sql2017ElasticPoolsClient
	ctx := meta.(*ArmClient).StopContext

	resGroup, serverName, name, err := parseArmSqlElasticPoolId(d.Id())
	if err != nil {
		return err
	}

	_, err = client.Delete(ctx, resGroup, serverName, name)

	return err
}

func parseArmSql2017ElasticPoolId(sqlElasticPoolId string) (string, string, string, error) {
	id, err := parseAzureResourceID(sqlElasticPoolId)
	if err != nil {
		return "", "", "", fmt.Errorf("[ERROR] Unable to parse SQL2017 ElasticPool ID %q: %+v", sqlElasticPoolId, err)
	}

	return id.ResourceGroup, id.Path["servers"], id.Path["elasticPools"], nil
}

func expandAzureRmSql2017ElasticPoolProperties(d *schema.ResourceData) *sql.ElasticPoolProperties {
	elasticPoolProperties := d.Get("elastic_pool_properties").([]interface{})
	elasticPoolProperty := elasticPoolProperties[0].(map[string]interface{})

	// state := elasticPoolProperty["state"].(string)
	// maxSizeBytes := int64(elasticPoolProperty["max_size_bytes"].(int))
	// zoneRedundant := elasticPoolProperty["zone_redundant"].(bool)
	// licenseType := elasticPoolProperty["license_type"].(string)

	perDatabaseSettings := elasticPoolProperty["per_database_settings"].([]interface{})
	perDatabaseSetting := perDatabaseSettings[0].(map[string]interface{})

	minCapacity := perDatabaseSetting["min_capacity"].(float64)
	maxCapacity := perDatabaseSetting["max_capacity"].(float64)

	elasticPoolPerDatabaseSettings := &sql.ElasticPoolPerDatabaseSettings{
		MinCapacity: utils.Float(minCapacity),
		MaxCapacity: utils.Float(maxCapacity),
	}

	props := &sql.ElasticPoolProperties{
		// State: sql.ElasticPoolState(state),
		// MaxSizeBytes:        &maxSizeBytes,
		PerDatabaseSettings: elasticPoolPerDatabaseSettings,
		// ZoneRedundant:       utils.Bool(zoneRedundant),
		// LicenseType:         sql.ElasticPoolLicenseType(licenseType),
	}

	return props
}

func expandAzureRmSql2017ElasticPoolSku(d *schema.ResourceData) *sql.Sku {
	skus := d.Get("sku").([]interface{})
	sku := skus[0].(map[string]interface{})

	name := sku["name"].(string)
	size := sku["size"].(string)
	tier := sku["tier"].(string)
	family := sku["family"].(string)
	capacity := sku["capacity"].(int)

	return &sql.Sku{
		Name:     utils.String(name),
		Size:     utils.String(size),
		Tier:     utils.String(tier),
		Family:   utils.String(family),
		Capacity: utils.Int32(int32(capacity)),
	}
}

func flattenAzureRmSql2017ElasticPoolSku(resp *sql.Sku) []interface{} {
	values := map[string]interface{}{}

	if name := resp.Name; name != nil {
		values["name"] = *name
	}

	if size := resp.Size; size != nil {
		values["size"] = *size
	}

	values["tier"] = *resp.Tier

	if family := resp.Family; family != nil {
		values["family"] = *family
	}

	if capacity := resp.Capacity; capacity != nil {
		values["capacity"] = *capacity
	}

	return []interface{}{values}
}

func flattenAzureRmSql2017ElasticPoolProperties(resp *sql.ElasticPoolProperties) []interface{} {
	elasticPoolProperty := map[string]interface{}{}

	if maxSizeBytes := resp.MaxSizeBytes; maxSizeBytes != nil {
		elasticPoolProperty["max_size_bytes"] = *maxSizeBytes
	}

	elasticPoolProperty["state"] = sql.ElasticPoolState(resp.State)

	if date := resp.CreationDate; date != nil {
		elasticPoolProperty["creation_date"] = date.String()
	}

	if zoneRedundant := resp.ZoneRedundant; zoneRedundant != nil {
		elasticPoolProperty["zone_redundant"] = *zoneRedundant
	}

	elasticPoolProperty["license_type"] = string(resp.LicenseType)

	elasticPoolProperty["per_database_settings"] = flattenAzureRmSql2017ElasticPoolPerDatabaseSettings(resp.PerDatabaseSettings)

	return []interface{}{elasticPoolProperty}
}

func flattenAzureRmSql2017ElasticPoolPerDatabaseSettings(resp *sql.ElasticPoolPerDatabaseSettings) []interface{} {
	perDatabaseSettings := map[string]interface{}{}

	if minCapacity := resp.MinCapacity; minCapacity != nil {
		perDatabaseSettings["min_capacity"] = *minCapacity
	}

	if maxCapacity := resp.MaxCapacity; maxCapacity != nil {
		perDatabaseSettings["max_capacity"] = *maxCapacity
	}

	return []interface{}{perDatabaseSettings}
}
