package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

var _ provider.Provider = &RabbitmqClientProvider{}

type RabbitmqClientProvider struct {
	version string
}

type ScaffoldingProviderModel struct {
	Uri types.String `tfsdk:"uri"`
}

func (p *RabbitmqClientProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "rmq"
	resp.Version = p.version
}

func (p *RabbitmqClientProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Terraform provider for RabbitMQ that enables AMQP-based message publishing and consumption.",
		Attributes: map[string]schema.Attribute{
			"uri": schema.StringAttribute{
				MarkdownDescription: "amqp URI",
				Optional:            true,
			},
		},
	}
}

func (p *RabbitmqClientProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ScaffoldingProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := amqp.Dial(data.Uri.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error connecting to RabbitMQ",
			"Could not create AMQP connection:\n\n"+err.Error(),
		)
		return
	}
	resp.DataSourceData = conn
}

func (p *RabbitmqClientProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *RabbitmqClientProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExampleDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RabbitmqClientProvider{
			version: version,
		}
	}
}
