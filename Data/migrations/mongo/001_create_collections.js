// ASGARD MongoDB Schema for Time-Series Data
// MongoDB 7+

// Use dedicated database
db = db.getSiblingDB("asgard");

// Satellite telemetry (high-frequency time-series)
db.createCollection("satellite_telemetry", {
    timeseries: {
        timeField: "timestamp",
        metaField: "satellite_id",
        granularity: "seconds"
    }
});

db.satellite_telemetry.createIndex({ "satellite_id": 1, "timestamp": -1 });

// Hunoid telemetry
db.createCollection("hunoid_telemetry", {
    timeseries: {
        timeField: "timestamp",
        metaField: "hunoid_id",
        granularity: "seconds"
    }
});

db.hunoid_telemetry.createIndex({ "hunoid_id": 1, "timestamp": -1 });

// Network flow logs (Sat_Net routing data)
db.createCollection("network_flows", {
    timeseries: {
        timeField: "timestamp",
        metaField: "source_node",
        granularity: "seconds"
    }
});

db.network_flows.createIndex({ "source_node": 1, "destination_node": 1, "timestamp": -1 });

// Giru security events
db.createCollection("security_events", {
    timeseries: {
        timeField: "timestamp",
        metaField: "event_type",
        granularity: "seconds"
    }
});

db.security_events.createIndex({ "event_type": 1, "severity": 1, "timestamp": -1 });

// VLA inference logs (for performance monitoring)
db.createCollection("vla_inferences");
db.vla_inferences.createIndex({ "hunoid_id": 1, "timestamp": -1 });
db.vla_inferences.createIndex({ "timestamp": -1 });

// AI router training data
db.createCollection("router_training_episodes");
db.router_training_episodes.createIndex({ "episode_id": 1 });
db.router_training_episodes.createIndex({ "timestamp": -1 });

print("MongoDB collections created successfully");
