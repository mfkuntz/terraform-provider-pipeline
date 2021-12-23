package pipelines

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// Project GET {{ host }}/access/api/v1/projects/{{prjKey}}/
//GET {{ host }}/artifactory/api/repositories/?prjKey={{prjKey}}
type PipelineSource struct {
	ProjectId int `json:"projectId"`
	//Project                   string          `json:"project"`
	ProjectIntegrationId int      `json:"projectIntegrationId"`
	RepositoryFullName   string   `json:"repositoryFullName"`
	Branch               string   `json:"branch"`
	FileFilter           string   `json:"fileFilter"`
	IsMultiBranch        bool     `json:"isMultiBranch"`
	BranchExcludePattern string   `json:"branchExcludePattern"`
	BranchIncludePattern string   `json:"branchIncludePattern"`
	Environments         []string `json:"environments"`
	TemplateId           int      `json:"templateId"`
	ID                   int      `json:"id,omitempty"`
}

const pipelineSourcesUrl = "pipelines/api/v1/pipelinesources"

// func verifyPipelineSource(id string, request *resty.Request) (*resty.Response, error) {
// 	return request.Head(pipelinesSourcesUrl + id)
// }

func pipelineSourceResource() *schema.Resource {

	var pipelineSourceSchema = map[string]*schema.Schema{
		"project_id": {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntAtLeast(0),
			Description:  "Id of the project where the pipeline source will live.",
		},
		"project_integration_id": {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntAtLeast(0),
			Description:  "Id of the project Github integration to use to create the pipeline source.",
		},
		"repository_full_name": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The full name of the Git repository including the user/organization as it appears in a Git clone command. For example, myOrg/myProject.",
		},
		"file_filter": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "A regular expression to determine which files to include in pipeline sync (the YML files), with default pipelines.yml. If a templateId was provided, it must be values.yml.",
		},
		"is_multi_branch": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "True if the pipeline source is to be a multi-branch pipeline source. Otherwise, it will be a single-branch pipeline source.",
		},
		"branch_exclude_pattern": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "For multi-branch pipeline sources, a regular expression of the branches to exclude.",
		},
		"branch_include_pattern": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "For multi-branch pipeline sources, a regular expression of the branches to include.",
		},
		"environments": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "In a project, an array of environment names in which this pipeline source will be.",
		},
		"template_id": {
			Type:         schema.TypeInt,
			Optional:     true,
			ValidateFunc: validation.IntAtLeast(0),
			Description:  "The id of a template to use for this pipeline source, in which case the fileFilter will only specify the values.yml",
		},
	}

	var unpackPipelineSource = func(data *schema.ResourceData) (PipelineSource, error) {
		d := &ResourceData{data}

		pipelineSource := PipelineSource{
			ProjectId:            d.getInt("project_id"),
			ProjectIntegrationId: d.getInt("projectIntegration_id"),
			RepositoryFullName:   d.getString("repository_full_name"),
			Branch:               d.getString("branch"),
			FileFilter:           d.getString("file_filter"),
			IsMultiBranch:        d.getBool("is_multi_branch"),
			BranchExcludePattern: d.getString("branch_exclude_pattern"),
			BranchIncludePattern: d.getString("branch_include_pattern"),
			Environments:         d.getList("environments"),
			TemplateId:           d.getInt("template_id"),
		}
		return pipelineSource, nil
	}

	var packPipelineSource = func(d *schema.ResourceData, pipelineSource PipelineSource) diag.Diagnostics {
		var errors []error
		setValue := mkLens(d)

		errors = setValue("project_id", pipelineSource.ProjectId)
		setValue("project_integration_id", pipelineSource.ProjectIntegrationId)
		setValue("repository_full_name", pipelineSource.RepositoryFullName)
		setValue("branch", pipelineSource.Branch)
		setValue("file_filter", pipelineSource.FileFilter)
		setValue("is_multi_branch", pipelineSource.IsMultiBranch)
		setValue("branch_exclude_pattern", pipelineSource.BranchExcludePattern)
		setValue("branch_include_pattern", pipelineSource.BranchIncludePattern)
		setValue("environments", pipelineSource.Environments)
		setValue("template_id", pipelineSource.TemplateId)

		if len(errors) > 0 {
			return diag.Errorf("failed to pack pipeline source %q", errors)
		}

		return nil
	}

	var readPipelineSource = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		pipelineSource := PipelineSource{}
		_, err := m.(*resty.Client).R().
			SetResult(&pipelineSource).
			Get(pipelineSourcesUrl + "/" + data.Id())
		if err != nil {
			return diag.FromErr(err)
		}
		return packPipelineSource(data, pipelineSource)
	}

	var createPipelineSource = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		log.Printf("[DEBUG] createPipelineSource")
		log.Printf("[TRACE] %+v\n", data)

		pipelineSource, err := unpackPipelineSource(data)
		if err != nil {
			return diag.FromErr(err)
		}

		resp, err := m.(*resty.Client).R().SetBody(pipelineSource).Post(pipelineSourcesUrl)
		if err != nil {
			return diag.FromErr(err)
		}
		var result PipelineSource
		err = json.Unmarshal(resp.Body(), &result)
		if err != nil {
			return diag.FromErr(err)
		}
		data.SetId(strconv.Itoa(result.ID))

		return readPipelineSource(ctx, data, m)
	}

	var updatePipelineSource = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		log.Printf("[DEBUG] updatePipelineSource")
		log.Printf("[TRACE] %+v\n", data)

		pipelineSource, err := unpackPipelineSource(data)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = m.(*resty.Client).R().
			SetBody(pipelineSource).
			Put(pipelineSourcesUrl + "/" + data.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		return readPipelineSource(ctx, data, m)
	}

	var deletePipelineSource = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		log.Printf("[DEBUG] deletePipelineSource")
		log.Printf("[TRACE] %+v\n", data)

		resp, err := m.(*resty.Client).R().
			Delete(pipelineSourcesUrl + "/" + data.Id())

		if err != nil && resp.StatusCode() == http.StatusNotFound {
			data.SetId("")
			return diag.FromErr(err)
		}

		return nil
	}

	return &schema.Resource{
		SchemaVersion: 1,
		CreateContext: createPipelineSource,
		ReadContext:   readPipelineSource,
		UpdateContext: updatePipelineSource,
		DeleteContext: deletePipelineSource,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema:      pipelineSourceSchema,
		Description: "Provides an Artifactory Pipeline Source resource.",
	}
}
