# Changelog

## 0.2.0

* Disambiguate same-named fields while keeping the original names. In the previous release, repeated fields had a suffix appended to the elements that had the same name anywhere in the schema. This prevented users from using the resource fields as seen in the API documentation (with the exception of fields using snake_case instead of camelCase to adhere to Terraform's naming conventions).  
With this release, the fields are named exactly as they appear in the API/CRD documentation.

## 0.1.0

Initial Beta release.
