CREATE TABLE synapse (
	id varchar(64) NOT NULL,
	org_id varchar(32) NOT NULL,
	name varchar(64), 
	is_active boolean NOT NULL DEFAULT TRUE,
	total_cpu_core float NOT NULL ,
	total_ram_mib int NOT NULL ,
	status enum('connected', 'disconnected') NOT NULL DEFAULT 'disconnected',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT pk_synapse_id PRIMARY KEY(id, org_id),
	CONSTRAINT fk_synapse_organizations_org_id FOREIGN KEY(org_id) REFERENCES organizations(id)
);
