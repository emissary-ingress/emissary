package v1

import (
	"context"

	"github.com/emissary-ingress/emissary/v3/internal/ir"
	"github.com/emissary-ingress/emissary/v3/internal/ir/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TranslateCachePolicies takes a list of kates unstructured objects and converts them to strongly typed
// CachePolicyContext IR for passing along to the xDS server.
func TranslateCachePolicies(ctx context.Context, unstructuredCachePolicies []*unstructured.Unstructured) []CachePolicyContext {

	cachePolicies := make([]CachePolicyContext, len(unstructuredCachePolicies))

	for _, cachePolicy := range unstructuredCachePolicies {

		cachePolicyMetadata := cachePolicy.UnstructuredContent()["metadata"]
		if cachePolicyMetadata == nil {
			// TODO: decide if we want to log or what...
			continue
		}

		metadata, ok := cachePolicyMetadata.(metav1.ObjectMeta)
		if !ok {
			continue
		}

		cachePolicySpec := cachePolicy.UnstructuredContent()["spec"]
		if cachePolicySpec == nil {
			continue
		}

		cachePolicyMap, ok := cachePolicySpec.(map[string]interface{})
		// We need to check if all these type assertions fail since we shouldnt rely on CRD validation to protect us from a panic state
		// I cant imagine a scenario where this would realisticly happen, but we generate a unique log message for tracability and skip processing it
		if !ok {
			// TODO: if this happens it is a programing error so at a minimum we should debug log it
			continue
		}

		cacheRules, ok := cachePolicyMap["rules"].([]map[string]interface{})
		if !ok {
			// TODO: this should only happen if CRD validation is not working thus a programming issue
			continue
		}

		rules := make([]CacheRuleContext, len(cacheRules))

		for _, cacheRule := range cacheRules {
			ruleContext := CacheRuleContext{
				Host:     cacheRule["host"].(string),
				Path:     cacheRule["path"].(string),
				CacheRef: ir.MapToNamespacedName(cacheRule["cacheRef"].(map[string]interface{})),
			}

			rules = append(rules, ruleContext)
		}

		cachePolicyContext := CachePolicyContext{
			NamespacedName: types.NamespacedName{
				Name:      metadata.Name,
				Namespace: metadata.Namespace,
			},
			Rules: rules,
		}

		cachePolicies = append(cachePolicies, cachePolicyContext)

	}

	return cachePolicies
}

// TranslateCaches takes a list of kates unstructured objects and converts them to strongly typed
// CacheContexts IR for passing along to the xDS server.
func TranslateCaches(ctx context.Context, unstructuredCaches []*unstructured.Unstructured) CacheMap {

	caches := CacheMap{}

	for _, unstructredCache := range unstructuredCaches {

		cacheMetadata := unstructredCache.UnstructuredContent()["metadata"]
		if cacheMetadata == nil {
			// TODO: decide if we want to log or what...
			continue
		}

		metadata, ok := cacheMetadata.(metav1.ObjectMeta)
		if !ok {
			continue
		}

		cacheSpec := unstructredCache.UnstructuredContent()["spec"]
		if cacheSpec == nil {
			continue
		}

		cacheSpecMap, ok := cacheSpec.(map[string]interface{})
		// We need to check if all these type assertions fail since we shouldnt rely on CRD validation to protect us from a panic state
		// I cant imagine a scenario where this would realisticly happen, but we generate a unique log message for tracability and skip processing it
		if !ok {
			// TODO: if this happens it is a programing error so at a minimum we should debug log it
			continue
		}

		providerType, ok := cacheSpecMap["cacheProvider"].(CacheProviderType)
		if !ok {
			// TODO: this should only happen if CRD validation is not working thus a programming issue
			continue
		}

		cacheContext := CacheContext{
			NamespacedName: types.NamespacedName{
				Name:      metadata.Name,
				Namespace: metadata.Namespace,
			},
			ProviderType: providerType,
		}
		cacheKey := cacheContext.NamespacedName.String()
		caches[cacheKey] = cacheContext
	}

	return caches
}
