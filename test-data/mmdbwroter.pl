#!/bin/perl

use MaxMind::DB::Writer::Tree;
 
# https://metacpan.org/pod/MaxMind::DB::Writer

my %types = (
    iso_code => 'utf8_string',
    country => 'map',
);
 
my $tree = MaxMind::DB::Writer::Tree->new(
    ip_version            => 4,
    record_size           => 24,
    database_type         => 'My-IP-Data',
    languages             => ['en'],
    description           => { en => 'My database of IP data' },
    map_key_type_callback => sub { $types{ $_[0] } },
);

my @data = (
    ["95.173.136.72/32", "RU"],
    ["192.229.221.103/32", "FR"],
    ["128.1.51.210/32", "CN"],
);

foreach my $row (@data) {
    my $net = $row->[0];
    my $iso_code = $row->[1];

    $tree->insert_network(
        $net,
        {
            country => {
                iso_code => $iso_code,
            },
        },
    );
}


open my $fh, '>:raw', 'geoip2-country.mmdb';
$tree->write_tree($fh);