package pkgs

import (
	"fmt"
	"strings"

	"org.commonjava/charon/module/config"
	"org.commonjava/charon/module/storage"
)

func IsMetadata(file string) bool {
	return isMVNMetadata(file) ||
		isNPMMetadata(file) ||
		strings.HasSuffix(file, "index.html")
}

func isMVNMetadata(file string) bool {
	return strings.HasSuffix(strings.TrimSpace(file), "maven-metadata.xml") ||
		strings.HasSuffix(strings.TrimSpace(file), "archetype-catalog.xml")
}

func isNPMMetadata(file string) bool {
	return strings.HasSuffix(strings.TrimSpace(file), "package.json")
}

func uploadPostProcess(
	failedFiles, failedMetas []string, productKey, bucket string) {
	postProcess(failedFiles, failedMetas, productKey, "uploaded to", bucket)
}

func rollbackPostProcess(
	failedFiles, failedMetas []string, productKey, bucket string) {
	postProcess(failedFiles, failedMetas, productKey, "rolled back from", bucket)
}

func postProcess(failedFiles, failedMetas []string, productKey, operation, bucket string) {
	if len(failedFiles) == 0 && len(failedMetas) == 0 {
		logger.Info(
			fmt.Sprintf("Product release %s is successfully %s Ronda service in bucket %s",
				productKey, operation, bucket))
	} else {
		total := len(failedFiles) + len(failedMetas)
		logger.Error(
			fmt.Sprintf("%d file(s) occur errors/warnings in bucket %s, please see errors.log for details.",
				total, bucket))
		logger.Error(
			fmt.Sprintf("Product release %s is %s Ronda service in bucket %s, but has some failures as below:",
				productKey, operation, bucket))
		if len(failedFiles) > 0 {
			logger.Error(fmt.Sprintf("Failed files: \n%s\n", failedFiles))
		}
		if len(failedMetas) > 0 {
			logger.Error(fmt.Sprintf("Failed metadata files: \n%s\n", failedMetas))
		}
	}
}

func invalidateCFPaths(cfClient *storage.CFCLient,
	target config.Target, invalidatePaths []string,
	root string, batchSize int) {

}

// def invalidate_cf_paths(
//     cf_client: CFClient,
//     bucket: Tuple[str, str, str, str, str],
//     invalidate_paths: List[str],
//     root="/",
//     batch_size=INVALIDATION_BATCH_DEFAULT
// ):
//     logger.info("Invalidating CF cache for %s", bucket[1])
//     bucket_name = bucket[1]
//     prefix = bucket[2]
//     prefix = "/" + prefix if not prefix.startswith("/") else prefix
//     domain = bucket[4]
//     slash_root = root
//     if not root.endswith("/"):
//         slash_root = slash_root + "/"
//     final_paths = []
//     for full_path in invalidate_paths:
//         path = full_path
//         if path.startswith(slash_root):
//             path = path[len(slash_root):]
//         if prefix:
//             path = os.path.join(prefix, path)
//         final_paths.append(path)
//     logger.debug("Invalidating paths: %s, size: %s", final_paths, len(final_paths))
//     if not domain:
//         domain = cf_client.get_domain_by_bucket(bucket_name)
//     if domain:
//         distr_id = cf_client.get_dist_id_by_domain(domain)
//         if distr_id:
//             real_batch_size = batch_size
//             for path in final_paths:
//                 if path.endswith('*'):
//                     real_batch_size = INVALIDATION_BATCH_WILDCARD
//                     break
//             result = cf_client.invalidate_paths(
//                 distr_id, final_paths, real_batch_size
//             )
//             if result:
//                 output = {}
//                 for invalidation in result:
//                     status = invalidation.get('Status')
//                     if status not in output:
//                         output[status] = []
//                     output[status].append(invalidation["Id"])
//                 non_completed = {}
//                 for status, ids in output.items():
//                     if status != INVALIDATION_STATUS_COMPLETED:
//                         non_completed[status] = ids
//                 logger.info(
//                     "The CF invalidating requests done, following requests "
//                     "are not completed yet:\n %s\nPlease use 'cf check' command to "
//                     "check its details.", non_completed
//                 )
//                 logger.debug(
//                     "All invalidations requested in this process:\n %s", output
//                 )
//     else:
//         logger.error(
//             "CF invalidating will not be performed because domain not found for"
//             " bucket %s. ", bucket_name
//         )
