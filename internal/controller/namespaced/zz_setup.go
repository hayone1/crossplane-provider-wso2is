// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/upjet/v2/pkg/controller"

	action "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/action/action"
	apiresource "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/application/apiresource"
	application "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/application/application"
	applicationshare "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/application/applicationshare"
	authorizedapi "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/application/authorizedapi"
	brandingpreference "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/branding/brandingpreference"
	brandingtext "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/branding/brandingtext"
	claim "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/claim/claim"
	claimdialect "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/claim/claimdialect"
	externalclaim "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/claim/externalclaim"
	oidcscope "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/claim/oidcscope"
	adminadvisorybanner "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/config/adminadvisorybanner"
	corsconfiguration "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/config/corsconfiguration"
	dcrconfiguration "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/config/dcrconfiguration"
	frauddetectionconfiguration "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/config/frauddetectionconfiguration"
	impersonationconfiguration "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/config/impersonationconfiguration"
	inboundprovisioningconfiguration "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/config/inboundprovisioningconfiguration"
	jwtkeyvalidatorconfiguration "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/config/jwtkeyvalidatorconfiguration"
	oauth2configuration "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/config/oauth2configuration"
	oauth2issuerusagescope "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/config/oauth2issuerusagescope"
	passivestsconfiguration "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/config/passivestsconfiguration"
	remotelogging "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/config/remotelogging"
	samlidpconfiguration "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/config/samlidpconfiguration"
	serverconfiguration "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/config/serverconfiguration"
	flow "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/flow/flow"
	governanceconnector "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/governance/governanceconnector"
	validationrules "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/governance/validationrules"
	group "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/identity/group"
	groupmembers "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/identity/groupmembers"
	role "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/identity/role"
	roleassignment "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/identity/roleassignment"
	user "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/identity/user"
	customauthenticator "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/idp/customauthenticator"
	identityprovider "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/idp/identityprovider"
	identityverificationprovider "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/idp/identityverificationprovider"
	emailsender "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/notification/emailsender"
	notificationtemplate "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/notification/notificationtemplate"
	notificationtemplatetype "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/notification/notificationtemplatetype"
	pushsender "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/notification/pushsender"
	smssender "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/notification/smssender"
	organization "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/organization/organization"
	organizationdiscovery "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/organization/organizationdiscovery"
	organizationdiscoveryconfiguration "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/organization/organizationdiscoveryconfiguration"
	usershare "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/organization/usershare"
	providerconfig "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/providerconfig"
	scriptlibrary "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/script/scriptlibrary"
	scriptsecret "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/script/scriptsecret"
	tenant "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/tenant/tenant"
	userstore "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/userstore/userstore"
	vctemplate "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/vc/vctemplate"
	webhook "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/webhook/webhook"
	workflow "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/workflow/workflow"
	workflowassociation "github.com/hayone1/crossplane-provider-wso2is/internal/controller/namespaced/workflow/workflowassociation"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		action.Setup,
		apiresource.Setup,
		application.Setup,
		applicationshare.Setup,
		authorizedapi.Setup,
		brandingpreference.Setup,
		brandingtext.Setup,
		claim.Setup,
		claimdialect.Setup,
		externalclaim.Setup,
		oidcscope.Setup,
		adminadvisorybanner.Setup,
		corsconfiguration.Setup,
		dcrconfiguration.Setup,
		frauddetectionconfiguration.Setup,
		impersonationconfiguration.Setup,
		inboundprovisioningconfiguration.Setup,
		jwtkeyvalidatorconfiguration.Setup,
		oauth2configuration.Setup,
		oauth2issuerusagescope.Setup,
		passivestsconfiguration.Setup,
		remotelogging.Setup,
		samlidpconfiguration.Setup,
		serverconfiguration.Setup,
		flow.Setup,
		governanceconnector.Setup,
		validationrules.Setup,
		group.Setup,
		groupmembers.Setup,
		role.Setup,
		roleassignment.Setup,
		user.Setup,
		customauthenticator.Setup,
		identityprovider.Setup,
		identityverificationprovider.Setup,
		emailsender.Setup,
		notificationtemplate.Setup,
		notificationtemplatetype.Setup,
		pushsender.Setup,
		smssender.Setup,
		organization.Setup,
		organizationdiscovery.Setup,
		organizationdiscoveryconfiguration.Setup,
		usershare.Setup,
		providerconfig.Setup,
		scriptlibrary.Setup,
		scriptsecret.Setup,
		tenant.Setup,
		userstore.Setup,
		vctemplate.Setup,
		webhook.Setup,
		workflow.Setup,
		workflowassociation.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupGated creates all controllers with the supplied logger and adds them to
// the supplied manager gated.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		action.SetupGated,
		apiresource.SetupGated,
		application.SetupGated,
		applicationshare.SetupGated,
		authorizedapi.SetupGated,
		brandingpreference.SetupGated,
		brandingtext.SetupGated,
		claim.SetupGated,
		claimdialect.SetupGated,
		externalclaim.SetupGated,
		oidcscope.SetupGated,
		adminadvisorybanner.SetupGated,
		corsconfiguration.SetupGated,
		dcrconfiguration.SetupGated,
		frauddetectionconfiguration.SetupGated,
		impersonationconfiguration.SetupGated,
		inboundprovisioningconfiguration.SetupGated,
		jwtkeyvalidatorconfiguration.SetupGated,
		oauth2configuration.SetupGated,
		oauth2issuerusagescope.SetupGated,
		passivestsconfiguration.SetupGated,
		remotelogging.SetupGated,
		samlidpconfiguration.SetupGated,
		serverconfiguration.SetupGated,
		flow.SetupGated,
		governanceconnector.SetupGated,
		validationrules.SetupGated,
		group.SetupGated,
		groupmembers.SetupGated,
		role.SetupGated,
		roleassignment.SetupGated,
		user.SetupGated,
		customauthenticator.SetupGated,
		identityprovider.SetupGated,
		identityverificationprovider.SetupGated,
		emailsender.SetupGated,
		notificationtemplate.SetupGated,
		notificationtemplatetype.SetupGated,
		pushsender.SetupGated,
		smssender.SetupGated,
		organization.SetupGated,
		organizationdiscovery.SetupGated,
		organizationdiscoveryconfiguration.SetupGated,
		usershare.SetupGated,
		providerconfig.SetupGated,
		scriptlibrary.SetupGated,
		scriptsecret.SetupGated,
		tenant.SetupGated,
		userstore.SetupGated,
		vctemplate.SetupGated,
		webhook.SetupGated,
		workflow.SetupGated,
		workflowassociation.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupWebhookWithManager registers conversion webhooks for all resource kinds in the group.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		action.SetupWebhookWithManager,
		apiresource.SetupWebhookWithManager,
		application.SetupWebhookWithManager,
		applicationshare.SetupWebhookWithManager,
		authorizedapi.SetupWebhookWithManager,
		brandingpreference.SetupWebhookWithManager,
		brandingtext.SetupWebhookWithManager,
		claim.SetupWebhookWithManager,
		claimdialect.SetupWebhookWithManager,
		externalclaim.SetupWebhookWithManager,
		oidcscope.SetupWebhookWithManager,
		adminadvisorybanner.SetupWebhookWithManager,
		corsconfiguration.SetupWebhookWithManager,
		dcrconfiguration.SetupWebhookWithManager,
		frauddetectionconfiguration.SetupWebhookWithManager,
		impersonationconfiguration.SetupWebhookWithManager,
		inboundprovisioningconfiguration.SetupWebhookWithManager,
		jwtkeyvalidatorconfiguration.SetupWebhookWithManager,
		oauth2configuration.SetupWebhookWithManager,
		oauth2issuerusagescope.SetupWebhookWithManager,
		passivestsconfiguration.SetupWebhookWithManager,
		remotelogging.SetupWebhookWithManager,
		samlidpconfiguration.SetupWebhookWithManager,
		serverconfiguration.SetupWebhookWithManager,
		flow.SetupWebhookWithManager,
		governanceconnector.SetupWebhookWithManager,
		validationrules.SetupWebhookWithManager,
		group.SetupWebhookWithManager,
		groupmembers.SetupWebhookWithManager,
		role.SetupWebhookWithManager,
		roleassignment.SetupWebhookWithManager,
		user.SetupWebhookWithManager,
		customauthenticator.SetupWebhookWithManager,
		identityprovider.SetupWebhookWithManager,
		identityverificationprovider.SetupWebhookWithManager,
		emailsender.SetupWebhookWithManager,
		notificationtemplate.SetupWebhookWithManager,
		notificationtemplatetype.SetupWebhookWithManager,
		pushsender.SetupWebhookWithManager,
		smssender.SetupWebhookWithManager,
		organization.SetupWebhookWithManager,
		organizationdiscovery.SetupWebhookWithManager,
		organizationdiscoveryconfiguration.SetupWebhookWithManager,
		usershare.SetupWebhookWithManager,
		providerconfig.SetupWebhookWithManager,
		scriptlibrary.SetupWebhookWithManager,
		scriptsecret.SetupWebhookWithManager,
		tenant.SetupWebhookWithManager,
		userstore.SetupWebhookWithManager,
		vctemplate.SetupWebhookWithManager,
		webhook.SetupWebhookWithManager,
		workflow.SetupWebhookWithManager,
		workflowassociation.SetupWebhookWithManager,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
