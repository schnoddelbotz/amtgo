package database

var mysqlSchema = `

SET foreign_key_checks = 0;

-- notifications: Short messages for dashboard
CREATE TABLE notification (
  id                INTEGER NOT NULL AUTO_INCREMENT PRIMARY KEY,
  tstamp            INTEGER,
  user_id           INT          NOT NULL,
  ntype             VARCHAR(12),
  message           VARCHAR(64),

  FOREIGN KEY(user_id) REFERENCES user(id)
);
CREATE TRIGGER tstampTrigger BEFORE INSERT ON notification FOR EACH ROW SET new.tstamp = UNIX_TIMESTAMP(NOW());

-- organizational units / rooms
CREATE TABLE ou (
  id                INTEGER      NOT NULL AUTO_INCREMENT PRIMARY KEY,
  parent_id         INTEGER      NULL,
  optionset_id      INT,
  name              VARCHAR(128) NOT NULL,
  description       VARCHAR(255),
  idle_power        REAL,
  logging           INT          DEFAULT 1,

  FOREIGN KEY(optionset_id) REFERENCES optionset(id),
  FOREIGN KEY(parent_id) REFERENCES ou(id) ON DELETE RESTRICT
);

-- clients to be placed into ous
CREATE TABLE user (
  id                INTEGER      NOT NULL AUTO_INCREMENT PRIMARY KEY,
  ou_id             INTEGER      NOT NULL,    -- currently only one related (top) OU; no distinct permissions
  is_enabled        INTEGER      DEFAULT 1,
  is_admin          INTEGER      DEFAULT 1,
  can_control       INTEGER      DEFAULT 1,
  name              VARCHAR(64)  UNIQUE NOT NULL,
  fullname          VARCHAR(64)  NOT NULL,
  password          VARCHAR(64)  NOT NULL,
  passsalt          VARCHAR(64)  NOT NULL,

  FOREIGN KEY(ou_id) REFERENCES ou(id)
);

-- clients to be placed into ous
CREATE TABLE host (
  id                INTEGER      NOT NULL AUTO_INCREMENT PRIMARY KEY,
  ou_id             INTEGER      NOT NULL,
  hostname          VARCHAR(64)  NOT NULL,
  enabled           INTEGER      DEFAULT 1,

  FOREIGN KEY(ou_id) REFERENCES ou(id) ON DELETE RESTRICT
);

-- state logging of hosts. log occurs upon state change.
CREATE TABLE statelog (
  host_id           INTEGER      NOT NULL AUTO_INCREMENT PRIMARY KEY,
  state_begin       INTEGER,
  open_port         INTEGER      DEFAULT NULL,
  state_amt         INTEGER(1),
  state_http        INTEGER(2),

  FOREIGN KEY(host_id) REFERENCES host(id) ON DELETE CASCADE
);
CREATE TRIGGER timestampTrigger BEFORE INSERT ON statelog FOR EACH ROW SET new.state_begin = UNIX_TIMESTAMP(NOW());
CREATE INDEX logdata_ld ON statelog (state_begin);
CREATE INDEX logdata_pd ON statelog (host_id);

CREATE VIEW logday AS
  SELECT DISTINCT(date(from_unixtime(state_begin))) AS id
  FROM statelog;


-- amt(c) option sets
CREATE TABLE optionset (
  id                INTEGER      NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name              VARCHAR(128) NOT NULL,
  description       VARCHAR(128),
  sw_v5             INTEGER DEFAULT 0,
  sw_dash           INTEGER DEFAULT 1,
  sw_scan22         INTEGER DEFAULT 1,
  sw_scan3389       INTEGER DEFAULT 1,
  sw_usetls         INTEGER DEFAULT 0,
  sw_skipcertchk    INTEGER DEFAULT 0,
  opt_timeout       INTEGER DEFAULT 10,
  opt_passfile      VARCHAR(128),
  opt_cacertfile    VARCHAR(128)
);

-- monitoring / scheduled tasks / interactive jobs
CREATE TABLE job (
  id                INTEGER      NOT NULL AUTO_INCREMENT PRIMARY KEY,
  job_type          INTEGER,     -- 1=interactive, 2=scheduled, 3=monitor
  job_status        INTEGER      DEFAULT '0',
  user_id           INTEGER      NOT NULL,

  amtc_cmd          CHAR(1)      NOT NULL,  -- U/D/R/C
  amtc_delay        REAL,
  amtc_bootdevice   CHAR(1)      DEFAULT NULL, -- tbd; no support in amtc yet

  amtc_hosts        TEXT, -- now ids of hosts...? FIXME tbd
  ou_id             INTEGER, -- req'd to determine optionset; allow override?

  start_time        INTEGER(4)   DEFAULT NULL, -- start time at day; tbd= minutes?
  repeat_interval   INTEGER, -- minutes
  repeat_days       INTEGER, -- pow(2, getdate()[wday])
  last_started      INTEGER(4)   DEFAULT NULL,
  last_done         INTEGER(4)   DEFAULT NULL,
  proc_pid          INTEGER, -- process id of currently running job

  description       VARCHAR(32), -- to reference it e.g. in logs (insb. sched)
  FOREIGN KEY(ou_id) REFERENCES ou(id) ON DELETE CASCADE,
  FOREIGN KEY(user_id) REFERENCES user(id)
);


--
-- Minimal initial set of records to let amtc-web look ok after initial install
--

-- example OUs ...
INSERT INTO ou VALUES(1,NULL,NULL,'ROOT','root',0,0);
INSERT INTO ou VALUES(2,1,NULL,'Student labs','Computer rooms',0,0);
INSERT INTO ou VALUES(3,2,NULL,'E Floor','All rooms on E floor',0,0);
INSERT INTO ou VALUES(4,3,3,'E 19','An example room on E floor',24.5,1);

-- example notification that will show up in dashboard
INSERT INTO notification (user_id,ntype,message) values (1,'warning','Congrats, amtgo installed!');

-- some amtc option sets
INSERT INTO optionset VALUES(1,'DASH / No TLS','Uses DASH',0,1,1,1,0,0,10,'amtpassword.txt','');
INSERT INTO optionset VALUES(2,'DASH / TLS / VerifyCertSkip','Skips TLS certificate verification',0,1,1,1,1,1,10,'amtpassword.txt','');
INSERT INTO optionset VALUES(3,'DASH / TLS / VerifyCert','Most secure optionset',0,1,1,1,1,0,15,'amtpassword.txt','my.ca.crt');

-- put some hosts into E19
INSERT INTO host VALUES(1,4,'labpc-e19-01',1);
INSERT INTO host VALUES(2,4,'labpc-e19-02',1);
INSERT INTO host VALUES(3,4,'labpc-e19-03',1);
INSERT INTO host VALUES(4,4,'labpc-e19-04',1);
INSERT INTO host VALUES(5,4,'labpc-e19-05',1);
INSERT INTO host VALUES(6,4,'labpc-e19-06',1);
INSERT INTO host VALUES(7,4,'labpc-e19-07',1);
INSERT INTO host VALUES(8,4,'labpc-e19-08',1);
INSERT INTO host VALUES(9,4,'labpc-e19-09',1);
INSERT INTO host VALUES(10,4,'labpc-e19-10',1);
INSERT INTO host VALUES(11,4,'labpc-e19-11',1);
INSERT INTO host VALUES(12,4,'labpc-e19-12',1);
INSERT INTO host VALUES(13,4,'labpc-e19-13',1);
INSERT INTO host VALUES(14,4,'labpc-e19-14',1);
INSERT INTO host VALUES(15,4,'labpc-e19-15',1);


INSERT INTO job VALUES(1,2,0,1,'U',2.5,NULL,NULL,4,480,NULL,127,NULL,NULL,NULL,'Power-Up E19 Mon-Sun');
INSERT INTO job VALUES(2,2,0,1,'D',1.0,NULL,NULL,4,1290,NULL,62,NULL,NULL,NULL,'Power-Down E19 Mon-Fri');
INSERT INTO job VALUES(3,2,0,1,'D',1.0,NULL,NULL,4,960,NULL,65,NULL,NULL,NULL,'Power-Down E19 Sat+Sun');
`
