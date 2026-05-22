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
    rse_names = [name["rse"] for name in rses]
    rse_names.sort()

    static_space = []
    used_space = []
    fraction_space = []
    free_space = []
    rse_names_final = []

    for rse in rse_names:
        site_usage = list(client.get_rse_usage(rse))
        sources = {el["source"]: el["used"] for el in site_usage if el["source"] in ["static", "rucio"]}
        if len(sources) == 2:  # checking if both rucio and static info exists
            static_space.append(sources["static"] * 1e-12)
            used_space.append(sources["rucio"] * 1e-12)
            fraction_space.append(100. * used_space[-1] / static_space[-1])
            free_space.append(static_space[-1] - used_space[-1])
            rse_names_final.append(rse)

    # print(f"rse_names: {rse_names_final},\nlen: {len(rse_names)}\n")
    # print(f"fraction_space: {fraction_space},\nlen: {len(fraction_space)}\n")
    # print(f"free_space: {free_space},\nlen: {len(free_space)}\n")
    # print(f"static_space: {static_space},\nlen: {len(static_space)}\n")
    # print(f"used_space: {used_space},\nlen: {len(used_space)}\n")

    data = {'RSE': rse_names_final, '   Used space   ': used_space, '   Total space   ': static_space,
            '   Fraction used (%)   ': fraction_space, '   Free space   ': free_space}

    df = pd.DataFrame.from_dict(data=data).round(1)
    # df.style.set_properties(subset=['   Used space   '], **{'width': '300px'})
    html = df.to_html(escape=False, index=False, col_space='150px')
    html = html.replace(
        'table border="1" class="dataframe"',
        'table id="dataframe" class="display compact" style="width:100%;"',
    )
    html = html.replace('style="text-align: right;"', "")

    with open(os.path.join(template_dir, "htmltemplate.html")) as f:
        htm_template = f.read()

    current_date = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
    main_html = htm_template.replace("XXX", current_date)
    main_html = main_html.replace("____MAIN_BLOCK____", html)

    with open(output, "w+") as f:
        f.write(main_html)


if __name__ == "__main__":
    main()
