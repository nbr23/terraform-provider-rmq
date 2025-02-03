package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

var _ datasource.DataSource = &MessageDataSource{}

func NewExampleDataSource() datasource.DataSource {
	return &MessageDataSource{}
}

type MessageDataSource struct {
	client *amqp.Connection
}

type MessageDataSourceModel struct {
	QueueName       types.String `tfsdk:"queue_name"`
	QueueNamePrefix types.String `tfsdk:"queue_name_prefix"`
	Ack             types.Bool   `tfsdk:"ack"`
	MessageCount    types.Int64  `tfsdk:"message_count"`
	Exchange        types.String `tfsdk:"exchange"`
	RoutingKey      types.String `tfsdk:"routing_key"`
	Timeout         types.String `tfsdk:"timeout"`
	Messages        []Message    `tfsdk:"messages"`
}

type Message struct {
	Body       types.String            `tfsdk:"body"`
	Headers    map[string]types.String `tfsdk:"headers"`
	RoutingKey types.String            `tfsdk:"routing_key"`

	ContentType     types.String `tfsdk:"content_type"`
	ContentEncoding types.String `tfsdk:"content_encoding"`
	DeliveryMode    uint8        `tfsdk:"delivery_mode"`
	Priority        uint8        `tfsdk:"priority"`
	CorrelationId   types.String `tfsdk:"correlation_id"`
	ReplyTo         types.String `tfsdk:"reply_to"`
	Timestamp       types.Int64  `tfsdk:"timestamp"`
	Type            types.String `tfsdk:"type"`
	UserId          types.String `tfsdk:"user_id"`
	AppId           types.String `tfsdk:"app_id"`
}

func (d *MessageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_message"
}

func (d *MessageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Consume `message_count` messages from `exchange`.\nA temporary queue (either named `queue_name`, or `queue_name_prefix+uuid()`) and binds it to `exchange` using the specified routing key.\nOnce `message_count` messages have been consumed successfully, the temporary queue is deleted.",
		Attributes: map[string]schema.Attribute{
			"queue_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the temporary queue to create for message consumption",
			},
			"queue_name_prefix": schema.StringAttribute{
				Optional:    true,
				Description: "Name prefix of the temporary queue to create for message consumption",
			},
			"ack": schema.BoolAttribute{
				Required:    true,
				Description: "Whether to acknowledge consumed messages",
			},
			"message_count": schema.Int64Attribute{
				Required:    true,
				Description: "Number of messages to consume",
			},
			"exchange": schema.StringAttribute{
				Required:    true,
				Description: "Exchange to bind and consume from",
			},
			"routing_key": schema.StringAttribute{
				Required:    true,
				Description: "Routing key to bind to the exchange",
			},
			"timeout": schema.StringAttribute{
				Optional:    true,
				Description: "Timeout for consuming messages",
			},
			"messages": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"body": schema.StringAttribute{
							Computed: true,
						},
						"headers": schema.MapAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						"routing_key": schema.StringAttribute{
							Computed: true,
						},
						"content_type": schema.StringAttribute{
							Computed: true,
						},
						"content_encoding": schema.StringAttribute{
							Computed: true,
						},
						"delivery_mode": schema.Int64Attribute{
							Computed: true,
						},
						"priority": schema.Int64Attribute{
							Computed: true,
						},
						"correlation_id": schema.StringAttribute{
							Computed: true,
						},
						"reply_to": schema.StringAttribute{
							Computed: true,
						},
						"timestamp": schema.Int64Attribute{
							Computed: true,
						},
						"type": schema.StringAttribute{
							Computed: true,
						},
						"user_id": schema.StringAttribute{
							Computed: true,
						},
						"app_id": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *MessageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*amqp.Connection)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *MessageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MessageDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var timeout time.Duration
	if data.Timeout.ValueString() != "" {
		var err error
		timeout, err = time.ParseDuration(data.Timeout.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid timeout", err.Error())
			return
		}
	} else {
		timeout = 20 * time.Minute
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ch, err := d.client.Channel()

	if err != nil {
		resp.Diagnostics.AddError("Error creating channel", err.Error())
		return
	}

	defer ch.Close()

	if data.QueueName.ValueString() != "" && data.QueueNamePrefix.ValueString() != "" {
		resp.Diagnostics.AddError("Error creating queue", "Only one of queue_name or queue_name_prefix can be set")
		return
	}

	if data.QueueName.ValueString() == "" {
		if data.QueueNamePrefix.ValueString() == "" {
			data.QueueNamePrefix = types.StringValue("tf-")
		}
		data.QueueName = types.StringValue(fmt.Sprintf("%s%s", data.QueueNamePrefix.ValueString(), uuid.New().String()))
	}

	queue, err := ch.QueueDeclare(
		data.QueueName.ValueString(), // name
		false,                        // durable
		true,                         // auto-delete
		true,                         // exclusive
		false,                        // no-wait
		nil,                          // arguments
	)

	if err != nil {
		resp.Diagnostics.AddError("Error declaring queue", err.Error())
		return
	}

	err = ch.QueueBind(
		queue.Name,
		data.RoutingKey.ValueString(),
		data.Exchange.ValueString(),
		false,
		nil,
	)

	if err != nil {
		resp.Diagnostics.AddError("Error binding queue", err.Error())
		return
	}

	msgs, err := ch.Consume(
		queue.Name,
		"",    // consumer
		false, // auto-ack
		true,  // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)

	if err != nil {
		resp.Diagnostics.AddError("Error consuming messages", err.Error())
		return
	}

	done := make(chan error)
	go func() {
		for i := 0; i < int(data.MessageCount.ValueInt64()); i++ {
			msg := <-msgs

			if msg.Body == nil {
				break
			}

			data.Messages = append(data.Messages, Message{
				Body:        types.StringValue(string(msg.Body)),
				Headers:     map[string]types.String{},
				RoutingKey:  types.StringValue(msg.RoutingKey),
				ContentType: types.StringValue(msg.ContentType),

				ContentEncoding: types.StringValue(msg.ContentEncoding),
				DeliveryMode:    msg.DeliveryMode,
				Priority:        msg.Priority,
				CorrelationId:   types.StringValue(msg.CorrelationId),
				ReplyTo:         types.StringValue(msg.ReplyTo),
				Timestamp:       types.Int64Value(msg.Timestamp.Unix()),
				Type:            types.StringValue(msg.Type),
				UserId:          types.StringValue(msg.UserId),
				AppId:           types.StringValue(msg.AppId),
			})

			if data.Ack.ValueBool() {
				err = msg.Ack(false)
				if err != nil {
					resp.Diagnostics.AddError("Error acknowledging message", err.Error())
					done <- err
					return
				}
			}
		}
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			return
		}
	case <-ctx.Done():
		resp.Diagnostics.AddError(
			"Timeout exceeded",
			fmt.Sprintf("Operation timed out after %s", timeout),
		)
		return
	}

	_, err = ch.QueueDelete(
		queue.Name,
		false, // ifUnused
		false, // ifEmpty
		false, // noWait
	)

	if err != nil {
		resp.Diagnostics.AddError("Error deleting queue", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
