import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AgGridAngular, AgGridModule } from 'ag-grid-angular';
import { ColDef, GridOptions } from 'ag-grid-community';
import {
  dateFilterComparator,
  dateSortComparator,
  User,
} from '@jdw/angular-usersui-util';
import { UsersService } from '@jdw/angular-usersui-data-access';
import { UsersActionsButtonCellRendererComponent } from '../users-actions-button-cell-renderer/users-actions-button-cell-renderer.component';

@Component({
  selector: 'lib-users',
  standalone: true,
  imports: [CommonModule, AgGridAngular, AgGridModule],
  templateUrl: './users.component.html',
  styleUrl: './users.component.scss',
})
export class UsersComponent implements OnInit {
  private usersService: UsersService = inject(UsersService);
  users: User[] = [];

  colDefs: ColDef[] = [
    { field: 'id', headerName: 'ID', cellDataType: 'number' },
    { field: 'emailAddress', headerName: 'Email', cellDataType: 'text' },
    { field: 'status', headerName: 'Status', cellDataType: 'text' },
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
      cellRenderer: UsersActionsButtonCellRendererComponent,
      maxWidth: 72,
      minWidth: 72,
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
  };

  ngOnInit() {
    this.usersService.getUsers().subscribe({
      next: (response) => {
        this.users = response;
      },
    });
  }
}
