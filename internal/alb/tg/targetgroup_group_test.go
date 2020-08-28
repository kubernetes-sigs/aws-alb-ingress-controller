package tg

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/kubernetes-sigs/aws-alb-ingress-controller/internal/aws"
	"github.com/kubernetes-sigs/aws-alb-ingress-controller/internal/ingress/controller/config"
	"github.com/kubernetes-sigs/aws-alb-ingress-controller/internal/ingress/controller/store"
	"github.com/kubernetes-sigs/aws-alb-ingress-controller/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type TGReconcileCall struct {
	// Ingress defaults to tc.Ingress
	Backend     extensions.IngressBackend
	TargetGroup TargetGroup
	Err         error
}

type DeleteTargetGroupByArnCall struct {
	Arn string
	Err error
}

func TestDefaultGroupController_Reconcile(t *testing.T) {
	for _, tc := range []struct {
		Name             string
		Ingress          extensions.Ingress
		TGReconcileCalls []TGReconcileCall
		TagTGGroupCall   *TagTGGroupCall
		ExpectedTGGroup  TargetGroupGroup
		ExpectedError    error
	}{
		{
			Name: "Reconcile succeeds with duplicated targetGroup",
			Ingress: extensions.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress",
					Namespace: "namespace",
				},
				Spec: extensions.IngressSpec{
					Rules: []extensions.IngressRule{
						{
							Host: "d1.example.com",
							IngressRuleValue: extensions.IngressRuleValue{
								HTTP: &extensions.HTTPIngressRuleValue{
									Paths: []extensions.HTTPIngressPath{
										{
											Path: "/path1",
											Backend: extensions.IngressBackend{
												ServiceName: "service1",
												ServicePort: intstr.FromInt(80),
											},
										},
										{
											Path: "/path2",
											Backend: extensions.IngressBackend{
												ServiceName: "service1",
												ServicePort: intstr.FromInt(443),
											},
										},
									},
								},
							},
						},
						{
							Host: "d2.example.com",
							IngressRuleValue: extensions.IngressRuleValue{
								HTTP: &extensions.HTTPIngressRuleValue{
									Paths: []extensions.HTTPIngressPath{
										{
											Path: "/path1",
											Backend: extensions.IngressBackend{
												ServiceName: "service1",
												ServicePort: intstr.FromInt(80),
											},
										},
										{
											Path: "/path2",
											Backend: extensions.IngressBackend{
												ServiceName: "service2",
												ServicePort: intstr.FromInt(443),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TGReconcileCalls: []TGReconcileCall{
				{
					Backend: extensions.IngressBackend{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					},
					TargetGroup: TargetGroup{
						Arn: "arn1",
					},
				},
				{
					Backend: extensions.IngressBackend{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(443),
					},
					TargetGroup: TargetGroup{
						Arn: "arn2",
					},
				},
				{
					Backend: extensions.IngressBackend{
						ServiceName: "service2",
						ServicePort: intstr.FromInt(443),
					},
					TargetGroup: TargetGroup{
						Arn: "arn3",
					},
				},
			},
			TagTGGroupCall: &TagTGGroupCall{
				Namespace:   "namespace",
				IngressName: "ingress",
				Tags:        map[string]string{"key1": "value1", "key2": "value2"},
			},
			ExpectedTGGroup: TargetGroupGroup{
				TGByBackend: map[extensions.IngressBackend]TargetGroup{
					{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					}: {Arn: "arn1"},
					{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(443),
					}: {Arn: "arn2"},
					{
						ServiceName: "service2",
						ServicePort: intstr.FromInt(443),
					}: {Arn: "arn3"},
				},
				selector: map[string]string{"key1": "value1", "key2": "value2"},
			},
		},
		{
			Name: "Reconcile succeeds with empty HTTP rule",
			Ingress: extensions.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress",
					Namespace: "namespace",
				},
				Spec: extensions.IngressSpec{
					Rules: []extensions.IngressRule{
						{
							Host: "d1.example.com",
							IngressRuleValue: extensions.IngressRuleValue{
								HTTP: &extensions.HTTPIngressRuleValue{
									Paths: []extensions.HTTPIngressPath{
										{
											Path: "/path1",
											Backend: extensions.IngressBackend{
												ServiceName: "service1",
												ServicePort: intstr.FromInt(80),
											},
										},
										{
											Path: "/path2",
											Backend: extensions.IngressBackend{
												ServiceName: "service1",
												ServicePort: intstr.FromInt(443),
											},
										},
									},
								},
							},
						},
						{
							Host: "d2.example.com",
						},
					},
				},
			},
			TGReconcileCalls: []TGReconcileCall{
				{
					Backend: extensions.IngressBackend{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					},
					TargetGroup: TargetGroup{
						Arn: "arn1",
					},
				},
				{
					Backend: extensions.IngressBackend{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(443),
					},
					TargetGroup: TargetGroup{
						Arn: "arn2",
					},
				},
			},
			TagTGGroupCall: &TagTGGroupCall{
				Namespace:   "namespace",
				IngressName: "ingress",
				Tags:        map[string]string{"key1": "value1", "key2": "value2"},
			},
			ExpectedTGGroup: TargetGroupGroup{
				TGByBackend: map[extensions.IngressBackend]TargetGroup{
					{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					}: {Arn: "arn1"},
					{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(443),
					}: {Arn: "arn2"},
				},
				selector: map[string]string{"key1": "value1", "key2": "value2"},
			},
		},
		{
			Name: "Reconcile succeeds with default backend",
			Ingress: extensions.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress",
					Namespace: "namespace",
				},
				Spec: extensions.IngressSpec{
					Backend: &extensions.IngressBackend{
						ServiceName: "service2",
						ServicePort: intstr.FromInt(443),
					},
					Rules: []extensions.IngressRule{
						{
							IngressRuleValue: extensions.IngressRuleValue{
								HTTP: &extensions.HTTPIngressRuleValue{
									Paths: []extensions.HTTPIngressPath{
										{
											Path: "/path1",
											Backend: extensions.IngressBackend{
												ServiceName: "service1",
												ServicePort: intstr.FromInt(80),
											},
										},
										{
											Path: "/path2",
											Backend: extensions.IngressBackend{
												ServiceName: "service1",
												ServicePort: intstr.FromInt(443),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TGReconcileCalls: []TGReconcileCall{
				{
					Backend: extensions.IngressBackend{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					},
					TargetGroup: TargetGroup{
						Arn: "arn1",
					},
				},
				{
					Backend: extensions.IngressBackend{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(443),
					},
					TargetGroup: TargetGroup{
						Arn: "arn2",
					},
				},
				{
					Backend: extensions.IngressBackend{
						ServiceName: "service2",
						ServicePort: intstr.FromInt(443),
					},
					TargetGroup: TargetGroup{
						Arn: "arn3",
					},
				},
			},
			TagTGGroupCall: &TagTGGroupCall{
				Namespace:   "namespace",
				IngressName: "ingress",
				Tags:        map[string]string{"key1": "value1", "key2": "value2"},
			},
			ExpectedTGGroup: TargetGroupGroup{
				TGByBackend: map[extensions.IngressBackend]TargetGroup{
					{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					}: {Arn: "arn1"},
					{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(443),
					}: {Arn: "arn2"},
					{
						ServiceName: "service2",
						ServicePort: intstr.FromInt(443),
					}: {Arn: "arn3"},
				},
				selector: map[string]string{"key1": "value1", "key2": "value2"},
			},
		},
		{
			Name: "Reconcile succeeds with backend using annotation",
			Ingress: extensions.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "ingress",
					Namespace:   "namespace",
					Annotations: map[string]string{},
				},
				Spec: extensions.IngressSpec{
					Rules: []extensions.IngressRule{
						{
							IngressRuleValue: extensions.IngressRuleValue{
								HTTP: &extensions.HTTPIngressRuleValue{
									Paths: []extensions.HTTPIngressPath{
										{
											Path: "/path1",
											Backend: extensions.IngressBackend{
												ServiceName: "service1",
												ServicePort: intstr.FromInt(80),
											},
										},
										{
											Path: "/path2",
											Backend: extensions.IngressBackend{
												ServiceName: "my-redirect",
												ServicePort: intstr.FromString("use-annotation"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TGReconcileCalls: []TGReconcileCall{
				{
					Backend: extensions.IngressBackend{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					},
					TargetGroup: TargetGroup{
						Arn: "arn1",
					},
				},
			},
			TagTGGroupCall: &TagTGGroupCall{
				Namespace:   "namespace",
				IngressName: "ingress",
				Tags:        map[string]string{"key1": "value1", "key2": "value2"},
			},
			ExpectedTGGroup: TargetGroupGroup{
				TGByBackend: map[extensions.IngressBackend]TargetGroup{
					{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					}: {Arn: "arn1"},
				},
				selector: map[string]string{"key1": "value1", "key2": "value2"},
			},
		},
		{
			Name: "Reconcile succeeds with service backend using annotation",
			Ingress: extensions.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress",
					Namespace: "namespace",
					Annotations: map[string]string{
						"alb.ingress.kubernetes.io/actions.weighted-routing": `{"Type":"forward","ForwardConfig":{"TargetGroups":[{"Weight":1,"ServiceName":"service1","ServicePort":"80"},{"Weight":1,"ServiceName":"service2","ServicePort":"80"}]}}`,
					},
				},
				Spec: extensions.IngressSpec{
					Rules: []extensions.IngressRule{
						{
							IngressRuleValue: extensions.IngressRuleValue{
								HTTP: &extensions.HTTPIngressRuleValue{
									Paths: []extensions.HTTPIngressPath{
										{
											Path: "/path1",
											Backend: extensions.IngressBackend{
												ServiceName: "service1",
												ServicePort: intstr.FromInt(80),
											},
										},
										{
											Path: "/path2",
											Backend: extensions.IngressBackend{
												ServiceName: "weighted-routing",
												ServicePort: intstr.FromString("use-annotation"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TGReconcileCalls: []TGReconcileCall{
				{
					Backend: extensions.IngressBackend{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					},
					TargetGroup: TargetGroup{
						Arn: "arn1",
					},
				},
				{
					Backend: extensions.IngressBackend{
						ServiceName: "service2",
						ServicePort: intstr.FromInt(80),
					},
					TargetGroup: TargetGroup{
						Arn: "arn2",
					},
				},
			},
			TagTGGroupCall: &TagTGGroupCall{
				Namespace:   "namespace",
				IngressName: "ingress",
				Tags:        map[string]string{"key1": "value1", "key2": "value2"},
			},
			ExpectedTGGroup: TargetGroupGroup{
				TGByBackend: map[extensions.IngressBackend]TargetGroup{
					{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					}: {Arn: "arn1"},
					{
						ServiceName: "service2",
						ServicePort: intstr.FromInt(80),
					}: {Arn: "arn2"},
				},
				selector: map[string]string{"key1": "value1", "key2": "value2"},
			},
		},
		{
			Name: "Reconcile failed when reconcile targetGroup",
			Ingress: extensions.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress",
					Namespace: "namespace",
				},
				Spec: extensions.IngressSpec{
					Rules: []extensions.IngressRule{
						{
							IngressRuleValue: extensions.IngressRuleValue{
								HTTP: &extensions.HTTPIngressRuleValue{
									Paths: []extensions.HTTPIngressPath{
										{
											Path: "/path1",
											Backend: extensions.IngressBackend{
												ServiceName: "service1",
												ServicePort: intstr.FromInt(80),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TGReconcileCalls: []TGReconcileCall{
				{
					Backend: extensions.IngressBackend{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					},
					Err: errors.New("TGReconcileCall"),
				},
			},
			ExpectedError: errors.New("TGReconcileCall"),
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			cloud := &mocks.CloudAPI{}
			mockNameTagGen := &MockNameTagGenerator{}
			if tc.TagTGGroupCall != nil {
				mockNameTagGen.On("TagTGGroup", tc.TagTGGroupCall.Namespace, tc.TagTGGroupCall.IngressName).Return(tc.TagTGGroupCall.Tags)
			}

			mockTGController := &MockController{}
			for _, call := range tc.TGReconcileCalls {
				mockTGController.On("Reconcile", mock.Anything, &tc.Ingress, call.Backend).Return(call.TargetGroup, call.Err)
			}

			mockStore := &store.MockStorer{}
			mockStore.On("GetConfig").Return(
				&config.Configuration{
					DefaultTargetType: elbv2.TargetTypeEnumInstance,
				}, nil)

			controller := &defaultGroupController{
				cloud:        cloud,
				nameTagGen:   mockNameTagGen,
				store:        mockStore,
				tgController: mockTGController,
			}

			tgGroup, err := controller.Reconcile(context.Background(), &tc.Ingress)
			assert.Equal(t, tc.ExpectedTGGroup, tgGroup)
			assert.Equal(t, tc.ExpectedError, err)
			cloud.AssertExpectations(t)
			mockNameTagGen.AssertExpectations(t)
			mockTGController.AssertExpectations(t)
		})
	}
}

func TestDefaultGroupController_GC(t *testing.T) {
	for _, tc := range []struct {
		Name                        string
		TGGroup                     TargetGroupGroup
		CurrentTargetGroups         []*elbv2.TargetGroup
		DeleteTargetGroupByArnCalls []DeleteTargetGroupByArnCall
		ExpectedError               error
	}{
		{
			Name: "GC succeeds",
			TGGroup: TargetGroupGroup{
				TGByBackend: map[extensions.IngressBackend]TargetGroup{
					{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					}: {Arn: "arn1"},
				},
			},
			CurrentTargetGroups: []*elbv2.TargetGroup{
				{TargetGroupArn: aws.String("arn1")},
				{TargetGroupArn: aws.String("arn2")},
				{TargetGroupArn: aws.String("arn3")},
			},
			DeleteTargetGroupByArnCalls: []DeleteTargetGroupByArnCall{
				{
					Arn: "arn2",
				},
				{
					Arn: "arn3",
				},
			},
		},
		{
			Name: "GC succeeds without deleting externalTargetArn even it's created by controller",
			TGGroup: TargetGroupGroup{
				TGByBackend: map[extensions.IngressBackend]TargetGroup{
					{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					}: {Arn: "arn1"},
				},
				externalTGARNs: []string{"arn3"},
			},
			CurrentTargetGroups: []*elbv2.TargetGroup{
				{TargetGroupArn: aws.String("arn1")},
				{TargetGroupArn: aws.String("arn2")},
				{TargetGroupArn: aws.String("arn3")},
			},
			DeleteTargetGroupByArnCalls: []DeleteTargetGroupByArnCall{
				{
					Arn: "arn2",
				},
			},
		},
		{
			Name: "GC failed when deleting targetGroup",
			TGGroup: TargetGroupGroup{
				TGByBackend: map[extensions.IngressBackend]TargetGroup{
					{
						ServiceName: "service1",
						ServicePort: intstr.FromInt(80),
					}: {Arn: "arn1"},
				},
			},
			CurrentTargetGroups: []*elbv2.TargetGroup{
				{TargetGroupArn: aws.String("arn1")},
				{TargetGroupArn: aws.String("arn2")},
				{TargetGroupArn: aws.String("arn3")},
			},
			DeleteTargetGroupByArnCalls: []DeleteTargetGroupByArnCall{
				{
					Arn: mock.Anything,
					Err: errors.New("DeleteTargetGroupByArnCall"),
				},
			},
			ExpectedError: errors.New("failed to delete targetGroup due to DeleteTargetGroupByArnCall"),
		},
	} {
		ctx := context.Background()
		cloud := &mocks.CloudAPI{}

		for _, call := range tc.DeleteTargetGroupByArnCalls {
			cloud.On("DeleteTargetGroupByArn", ctx, call.Arn).Return(call.Err)
		}
		mockNameTagGen := &MockNameTagGenerator{}
		mockTGController := &MockController{}
		for _, call := range tc.DeleteTargetGroupByArnCalls {
			mockTGController.On("StopReconcilingPodConditionStatus", call.Arn).Return()
		}

		controller := &defaultGroupController{
			cloud:        cloud,
			nameTagGen:   mockNameTagGen,
			tgController: mockTGController,
		}

		err := controller.GC(context.Background(), tc.CurrentTargetGroups, tc.TGGroup)
		assert.Equal(t, tc.ExpectedError, err)
		cloud.AssertExpectations(t)
		mockNameTagGen.AssertExpectations(t)
		mockTGController.AssertExpectations(t)
	}
}

func TestDefaultGroupController_Delete(t *testing.T) {
	for _, tc := range []struct {
		Name                        string
		CurrentTargetGroups         []*elbv2.TargetGroup
		DeleteTargetGroupByArnCalls []DeleteTargetGroupByArnCall
		ExpectedError               error
	}{
		{
			Name: "DELETE succeeds",
			CurrentTargetGroups: []*elbv2.TargetGroup{
				{TargetGroupArn: aws.String("arn1")},
				{TargetGroupArn: aws.String("arn2")},
				{TargetGroupArn: aws.String("arn3")},
			},
			DeleteTargetGroupByArnCalls: []DeleteTargetGroupByArnCall{
				{
					Arn: "arn1",
				},
				{
					Arn: "arn2",
				},
				{
					Arn: "arn3",
				},
			},
		},
		{
			Name: "DELETE failed when deleting targetGroup",
			CurrentTargetGroups: []*elbv2.TargetGroup{
				{TargetGroupArn: aws.String("arn1")},
				{TargetGroupArn: aws.String("arn2")},
				{TargetGroupArn: aws.String("arn3")},
			},
			DeleteTargetGroupByArnCalls: []DeleteTargetGroupByArnCall{
				{
					Arn: mock.Anything,
					Err: errors.New("DeleteTargetGroupByArnCall"),
				},
			},
			ExpectedError: errors.New("failed to delete targetGroup due to DeleteTargetGroupByArnCall"),
		},
	} {
		ctx := context.Background()
		cloud := &mocks.CloudAPI{}
		for _, call := range tc.DeleteTargetGroupByArnCalls {
			cloud.On("DeleteTargetGroupByArn", ctx, call.Arn).Return(call.Err)
		}
		mockNameTagGen := &MockNameTagGenerator{}
		mockTGController := &MockController{}
		for _, call := range tc.DeleteTargetGroupByArnCalls {
			mockTGController.On("StopReconcilingPodConditionStatus", call.Arn).Return()
		}

		controller := &defaultGroupController{
			cloud:        cloud,
			nameTagGen:   mockNameTagGen,
			tgController: mockTGController,
		}

		err := controller.Delete(context.Background(), tc.CurrentTargetGroups)
		assert.Equal(t, tc.ExpectedError, err)
		cloud.AssertExpectations(t)
		mockNameTagGen.AssertExpectations(t)
		mockTGController.AssertExpectations(t)
	}
}
