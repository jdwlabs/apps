import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AgGridAngular } from 'ag-grid-angular';
import { RolesService } from '@jdw/angular-shared-data-access';
import { Role } from '@jdw/angular-shared-util';
import { ColDef, GridOptions } from 'ag-grid-community';
import {
  dateFilterComparator,
  dateSortComparator,
} from '@jdw/angular-usersui-util';
import { RolesActionButtonCellRendererComponent } from '../roles-action-button-cell-renderer/roles-action-button-cell-renderer.component';

@Component({
  selector: 'lib-roles',
  imports: [CommonModule, AgGridAngular],
  templateUrl: './roles.component.html',
  styleUrl: './roles.component.scss',
})
export class RolesComponent implements OnInit {
  private rolesService: RolesService = inject(RolesService);
  loading = true;
  roles: Role[] = [];

  colDefs: ColDef[] = [
    { field: 'id', headerName: 'ID', cellDataType: 'number' },
    { field: 'name', headerName: 'Name', cellDataType: 'text' },
    { field: 'description', headerName: 'Description', cellDataType: 'text' },
    { field: 'status', headerName: 'Status', cellDataType: 'text' },
    {
      field: 'users',
      headerName: 'Users Count',
      valueGetter: (params) => params.data.users?.length || 0,
      cellDataType: 'number',
    },
    {
      field: 'createdByUserId',
      headerName: 'Created By',
      cellDataType: 'number',
    },
    {
      field: 'createdTime',
      headerName: 'Created Time',
      filter: 'agDateColumnFilter',
      filterParams: {
        comparator: dateFilterComparator,
      },
      comparator: dateSortComparator,
      cellDataType: 'text',
    },
    {
      field: 'modifiedByUserId',
      headerName: 'Modified By',
      cellDataType: 'number',
    },
    {
      field: 'modifiedTime',
      headerName: 'Modified Time',
      filter: 'agDateColumnFilter',
      filterParams: {
        comparator: dateFilterComparator,
      },
      comparator: dateSortComparator,
      cellDataType: 'text',
    },
    {
      field: 'actions',
      headerName: 'Actions',
      cellRenderer: RolesActionButtonCellRendererComponent,
      maxWidth: 172,
      minWidth: 172,
      resizable: false,
      filter: false,
      sortable: false,
    },
  ];

  defaultColDef: ColDef = {
    sortable: true,
    filter: true,
    resizable: true,
    cellStyle: {
      display: 'flex',
      alignItems: 'center',
      whiteSpace: 'nowrap',
      textOverflow: 'ellipsis',
      overflow: 'hidden',
    },
  };

  gridOptions: GridOptions = {
    loading: this.loading,
    columnDefs: this.colDefs,
    defaultColDef: this.defaultColDef,
    pagination: true,
    rowSelection: {
      mode: 'multiRow',
      checkboxes: false,
      headerCheckbox: false,
      enableClickSelection: true,
    },
    animateRows: true,
    enableCellTextSelection: true,
    autoSizeStrategy: {
      type: 'fitCellContents',
    },
    suppressColumnVirtualisation: true,
  };

  ngOnInit(): void {
    this.rolesService.getRoles().subscribe({
      next: (response) => {
        this.roles = response;
        this.loading = false;
      },
    });
  }
}
