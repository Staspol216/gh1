package pvz_domain

import (
	"testing"
	"time"
)

func TestOrder_ApplyPackaging(t *testing.T) {
	type fields struct {
		ID             int64
		RecipientID    int64
		ExpirationDate time.Time
		DeliveredDate  *time.Time
		RefundedDate   *time.Time
		ReturnedDate   *time.Time
		Status         OrderStatus
		History        []OrderRecord
		Weight         float64
		Worth          float64
	}
	type args struct {
		packagingType      string
		additionalMembrana bool
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantWorth float64
		wantErr   bool
	}{
		{
			name: "applies box packaging",
			fields: fields{
				Weight: 20,
				Worth:  100,
			},
			args: args{
				packagingType: "box",
			},
			wantWorth: 120,
		},
		{
			name: "applies bag packaging",
			fields: fields{
				Weight: 10,
				Worth:  100,
			},
			args: args{
				packagingType: "bag",
			},
			wantWorth: 105,
		},
		{
			name: "applies membrana packaging",
			fields: fields{
				Weight: 25,
				Worth:  100,
			},
			args: args{
				packagingType: "membrana",
			},
			wantWorth: 101,
		},
		{
			name: "adds additional membrana to box packaging",
			fields: fields{
				Weight: 20,
				Worth:  100,
			},
			args: args{
				packagingType:      "box",
				additionalMembrana: true,
			},
			wantWorth: 121,
		},
		{
			name: "does not add additional membrana to membrana packaging",
			fields: fields{
				Weight: 25,
				Worth:  100,
			},
			args: args{
				packagingType:      "membrana",
				additionalMembrana: true,
			},
			wantWorth: 101,
		},
		{
			name: "uses box packaging by default",
			fields: fields{
				Weight: 20,
				Worth:  100,
			},
			args: args{
				packagingType: "unknown",
			},
			wantWorth: 120,
		},
		{
			name: "returns error and keeps worth unchanged when bag is too heavy",
			fields: fields{
				Weight: 10.01,
				Worth:  100,
			},
			args: args{
				packagingType: "bag",
			},
			wantWorth: 100,
			wantErr:   true,
		},
		{
			name: "returns error and keeps worth unchanged when box is too heavy",
			fields: fields{
				Weight: 20.01,
				Worth:  100,
			},
			args: args{
				packagingType: "box",
			},
			wantWorth: 100,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Order{
				ID:             tt.fields.ID,
				RecipientID:    tt.fields.RecipientID,
				ExpirationDate: tt.fields.ExpirationDate,
				DeliveredDate:  tt.fields.DeliveredDate,
				RefundedDate:   tt.fields.RefundedDate,
				ReturnedDate:   tt.fields.ReturnedDate,
				Status:         tt.fields.Status,
				History:        tt.fields.History,
				Weight:         tt.fields.Weight,
				Worth:          tt.fields.Worth,
			}
			if err := o.ApplyPackaging(tt.args.packagingType, tt.args.additionalMembrana); (err != nil) != tt.wantErr {
				t.Fatalf("ApplyPackaging() error = %v, wantErr %v", err, tt.wantErr)
			}
			if o.Worth != tt.wantWorth {
				t.Errorf("ApplyPackaging() worth = %v, want %v", o.Worth, tt.wantWorth)
			}
		})
	}
}
