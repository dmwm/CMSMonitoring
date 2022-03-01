# !/usr/bin/env python
# -*- coding: utf-8 -*-
# Author: Benedikt Maier <benedikt [DOT] maier AT cern [DOT] ch>
# Create html table for Rucio RSE quotas
import click
import os
from datetime import datetime

import pandas as pd
from rucio.client import Client


@click.command()
@click.option("--output", required=True, help="For example: /eos/.../www/test/test.html")
@click.option("--template_dir", required=True,
              help="Html directory for main html template. For example: ~/CMSMonitoring/src/html/rucio_quotas")
def main(output=None, template_dir=None):
    client = Client()

    # DISK, no tape
    # rse_type=DISK or TAPE
    # cms_type=real or test
    # tier<3, exclude T3
    # tier>0, exclude T0,
    # \(T2_US_Caltech_Ceph|T2_PL_Warsaw) (exclude)

    RSE_EXPRESSION = r"rse_type=DISK&cms_type=real&tier<3&tier>0\(T2_US_Caltech_Ceph|T2_PL_Warsaw)"

    rses = list(client.list_rses(rse_expression=RSE_EXPRESSION))
    rse_names = []
    for e in rses:
        rse_names.append(e["rse"])
    rse_names.sort()

    static_space = []
    used_space = []
    fraction_space = []
    free_space = []

    for rse in rse_names:
        site_usage = list(client.get_rse_usage(rse))

        for source in site_usage:
            if source["source"] == "static":
                total = source["used"]
                static_space.append(total * 1e-12)
            if source["source"] == "rucio":
                rucio = source["used"]
                used_space.append(rucio * 1e-12)

        fraction_space.append(100. * used_space[-1] / static_space[-1])
        free_space.append(static_space[-1] - used_space[-1])

    # print(rse_names)
    # print(fraction_space)
    # print(free_space)

    data = {'RSE': rse_names, '   Used space   ': used_space, '   Total space   ': static_space,
            '   Fraction used (%)   ': fraction_space, '   Free space   ': free_space}

    df = pd.DataFrame.from_dict(data=data).round(1)
    # df.style.set_properties(subset=['   Used space   '], **{'width': '300px'})
    html = df.to_html(escape=False, index=False, col_space='150px')
    html = html.replace(
        'table border="1" class="dataframe"',
        'table id="dataframe" class="display compact" style="width:100%;"',
    )
    html = html.replace('style="text-align: right;"', "")

    #
    with open(os.path.join(template_dir, "htmltemplate.html")) as f:
        htm_template = f.read()

    current_date = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
    main_html = htm_template.replace("XXX", current_date)
    main_html = main_html.replace("____MAIN_BLOCK____", html)
    #
    with open(output, "w+") as f:
        f.write(main_html)


if __name__ == "__main__":
    main()
