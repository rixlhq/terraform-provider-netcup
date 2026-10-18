# Import using domain ID and record UUID identifier:
terraform import netcup_clouddns_record.www '12345/59994952-e024-4871-b04d-546d66f25e73'

# Or import using human-friendly composite syntax (domain_id/name/type/rdata):
terraform import netcup_clouddns_record.www '12345/www/A/192.0.2.1'
